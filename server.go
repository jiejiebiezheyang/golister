package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed index.html
var templateFS embed.FS

// Config 服务器配置。
type Config struct {
	Port     int
	Dir      string
	Title    string
	PageSize int // 每页条目数（大目录性能优化），<=0 时使用默认 200
	Log      *log.Logger
}

// Server HTTP 处理器与共享目录上下文。
type Server struct {
	cfg      Config
	rootAbs  string // 共享目录绝对路径（词法）
	rootReal string // 共享目录真实路径（解析符号链接后）
	exeBase  string // 可执行文件名，列表里自动隐藏
	pageSize int
	tpl      *template.Template
	log      *log.Logger
}

// NewServer 创建服务器并预编译模板。
func NewServer(cfg Config) (*Server, error) {
	if cfg.Log == nil {
		cfg.Log = log.New(os.Stderr, "", log.LstdFlags)
	}
	abs, err := filepath.Abs(cfg.Dir)
	if err != nil {
		return nil, fmt.Errorf("解析共享目录失败: %w", err)
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("共享目录不可访问 %s: %w", abs, err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("共享路径不是目录: %s", abs)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("解析共享目录真实路径失败: %w", err)
	}

	exeBase := ""
	if exe, err := os.Executable(); err == nil {
		exeBase = filepath.Base(exe)
	}

	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}

	tplBytes, err := templateFS.ReadFile("index.html")
	if err != nil {
		return nil, fmt.Errorf("读取模板失败: %w", err)
	}
	tpl, err := template.New("index").Parse(string(tplBytes))
	if err != nil {
		return nil, fmt.Errorf("解析模板失败: %w", err)
	}

	return &Server{
		cfg:      cfg,
		rootAbs:  filepath.Clean(abs),
		rootReal: filepath.Clean(real),
		exeBase:  exeBase,
		pageSize: pageSize,
		tpl:      tpl,
		log:      cfg.Log,
	}, nil
}

// Handler 返回带路由的 http.Handler。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	return mux
}

// handleRoot 是所有请求的入口：解析安全路径后按 目录/文件 分发。
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "仅支持 GET/HEAD 请求", http.StatusMethodNotAllowed)
		return
	}

	rel, err := cleanURLPath(r.URL.EscapedPath())
	if err != nil {
		http.Error(w, "非法请求路径", http.StatusBadRequest)
		s.log.Printf("非法路径 %q: %v", r.URL.EscapedPath(), err)
		return
	}

	full, err := s.fullPath(rel)
	if err != nil {
		http.Error(w, "拒绝访问", http.StatusForbidden)
		s.log.Printf("路径越界 %q: %v", rel, err)
		return
	}

	info, err := os.Stat(full)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			http.Error(w, "文件或目录不存在", http.StatusNotFound)
		case os.IsPermission(err):
			http.Error(w, "没有访问权限", http.StatusForbidden)
		default:
			http.Error(w, "无法访问文件", http.StatusInternalServerError)
		}
		s.log.Printf("访问失败 %s: %v", full, err)
		return
	}

	if info.IsDir() {
		s.handleDir(w, r, rel, full)
		return
	}
	s.handleFile(w, r, rel, full, info)
}

// hide 决定列表/打包时是否隐藏某条目。
func (s *Server) hide(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	return s.exeBase != "" && name == s.exeBase
}

// handleDir 处理目录：?zip=1 触发打包下载，否则渲染列表页。
func (s *Server) handleDir(w http.ResponseWriter, r *http.Request, rel, full string) {
	q := r.URL.Query()

	if q.Get("zip") == "1" {
		s.handleZip(w, r, rel, full)
		return
	}

	// 规范化：目录 URL 一律以 "/" 结尾（避免相对链接歧义）
	if !strings.HasSuffix(r.URL.EscapedPath(), "/") {
		loc := r.URL.EscapedPath() + "/"
		if r.URL.RawQuery != "" {
			loc += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, loc, http.StatusMovedPermanently)
		return
	}

	sortKey := validSortKey(q.Get("sort"))
	order := validOrder(q.Get("order"))
	search := q.Get("q")

	hideFn := func(name string) bool { return s.hide(name) }
	entries, err := s.listDir(rel, hideFn, search)
	if err != nil {
		http.Error(w, "无法读取目录", http.StatusInternalServerError)
		s.log.Printf("读取目录失败 %s: %v", full, err)
		return
	}
	sortEntries(entries, sortKey, order)

	total := len(entries)
	pageCount := (total + s.pageSize - 1) / s.pageSize
	if pageCount < 1 {
		pageCount = 1
	}
	page := parsePage(q.Get("page"))
	if page > pageCount {
		page = pageCount
	}
	start := (page - 1) * s.pageSize
	end := start + s.pageSize
	if end > total {
		end = total
	}
	pageEntries := entries[start:end]
	if pageEntries == nil {
		pageEntries = []fileEntry{}
	}

	data := s.buildPageData(rel, pageEntries, pageDataArgs{
		page:      page,
		pageCount: pageCount,
		total:     total,
		sort:      sortKey,
		order:     order,
		search:    search,
	})

	var buf bytes.Buffer
	if err := s.tpl.Execute(&buf, data); err != nil {
		http.Error(w, "页面渲染失败", http.StatusInternalServerError)
		s.log.Printf("模板执行失败 %s: %v", full, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

// handleZip 流式打包当前目录。
func (s *Server) handleZip(w http.ResponseWriter, r *http.Request, rel, _ string) {
	name := zipBaseName(rel) + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	setContentDisposition(w, name)
	if r.Method == http.MethodHead {
		return
	}
	hideFn := func(name string) bool { return s.hide(name) }
	if err := s.streamZipDir(w, rel, hideFn); err != nil {
		// 流已经启动后无法再改状态码，仅记录
		s.log.Printf("打包失败 %s: %v", rel, err)
	}
}
