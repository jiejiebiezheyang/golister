package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

//go:embed index.html
var embeddedFiles embed.FS

type FileItem struct {
	Name       string
	Href       string
	Size       string
	IsDir      bool
	IconSVG    template.HTML
	ModTime    string
	Type       string
	CanPreview bool
}

type PageData struct {
	Title       string
	Files       []FileItem
	CurrentPath string
}

var (
	imageExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".webp": true, ".svg": true}
	textExts  = map[string]bool{".txt": true, ".md": true, ".json": true, ".xml": true, ".html": true, ".css": true, ".js": true, ".go": true, ".py": true, ".java": true, ".c": true, ".cpp": true, ".h": true, ".hpp": true, ".sh": true, ".bat": true, ".yaml": true, ".yml": true, ".toml": true, ".ini": true, ".conf": true, ".log": true}
	codeExts  = map[string]bool{".go": true, ".py": true, ".java": true, ".c": true, ".cpp": true, ".h": true, ".hpp": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true, ".rs": true, ".rb": true, ".php": true, ".swift": true, ".kt": true, ".scala": true}
)

func getFileIcon(filename string, isDir bool) template.HTML {
	if isDir {
		return `<svg xmlns="http://www.w3.org/2000/svg" fill="#4a8fd7" height="20" width="20" viewBox="0 0 24 24"><path d="M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/></svg>`
	}

	ext := strings.ToLower(filepath.Ext(filename))

	switch {
	case imageExts[ext]:
		return `<svg xmlns="http://www.w3.org/2000/svg" fill="#4caf50" height="20" width="20" viewBox="0 0 24 24"><path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/></svg>`
	case textExts[ext], codeExts[ext]:
		return `<svg xmlns="http://www.w3.org/2000/svg" fill="#ff9800" height="20" width="20" viewBox="0 0 24 24"><path d="M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z"/></svg>`
	case ext == ".pdf":
		return `<svg xmlns="http://www.w3.org/2000/svg" fill="#f44336" height="20" width="20" viewBox="0 0 24 24"><path d="M20 2H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm-8.5 7.5c0 .83-.67 1.5-1.5 1.5H9v2H7.5V7H10c.83 0 1.5.67 1.5 1.5v1zm5 2c0 .83-.67 1.5-1.5 1.5h-2.5V7H15c.83 0 1.5.67 1.5 1.5v3zm4-3H19v1h1.5V11H19v2h-1.5V7h3v1.5zM9 9.5h1v-1H9v1zM4 6H2v14c0 1.1.9 2 2 2h14v-2H4V6zm10 5.5h1v-3h-1v3z"/></svg>`
	case ext == ".zip" || ext == ".rar" || ext == ".7z" || ext == ".tar" || ext == ".gz":
		return `<svg xmlns="http://www.w3.org/2000/svg" fill="#9c27b0" height="20" width="20" viewBox="0 0 24 24"><path d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm-6 10H6v-2h8v2zm4-4H6v-2h12v2z"/></svg>`
	default:
		return `<svg xmlns="http://www.w3.org/2000/svg" fill="#787878" height="20" width="20" viewBox="0 0 24 24"><path d="M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6z"/></svg>`
	}
}

func getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch {
	case imageExts[ext]:
		return "image"
	case textExts[ext], codeExts[ext]:
		return "text"
	case ext == ".pdf":
		return "pdf"
	default:
		return "file"
	}
}

func canPreview(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return imageExts[ext] || textExts[ext] || codeExts[ext]
}

func formatModTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// 人类可读大小，清晰易懂
func humanReadableSize(size int64) string {
	const (
		_          = iota
		KB float64 = 1 << (10 * iota)
		MB
		GB
		TB
	)

	sz := float64(size)
	switch {
	case sz < KB:
		return fmt.Sprintf("%d B", size)
	case sz < MB:
		return fmt.Sprintf("%.1f KB", sz/KB)
	case sz < GB:
		return fmt.Sprintf("%.1f MB", sz/MB)
	case sz < TB:
		return fmt.Sprintf("%.1f GB", sz/GB)
	default:
		return fmt.Sprintf("%.1f TB", sz/TB)
	}
}

func main() {
	// 增加help参数，打印参数说明
	help := flag.Bool("help", false, "显示帮助信息 (Show help message)")

	port := flag.Int("port", 8080, "服务器端口(默认8080) Server port (default 8080)")
	dir := flag.String("dir", ".", "共享目录(默认当前目录) Shared directory (default current directory)")
	title := flag.String("title", "cyan静态资源站", "网页标题(默认'cyan静态资源站') Page title (default 'cyan静态资源站')")

	flag.Parse()

	// 如果传了-help参数，打印帮助信息并退出
	if *help {
		fmt.Print(`
Usage:
  -port   服务器端口(默认8080) / Server port (default 8080)
  -dir    共享目录(默认当前目录) / Shared directory (default current directory)
  -title  网页标题(默认'cyan静态资源站') / Page title (default 'cyan静态资源站')
  -help   显示帮助信息 / Show help message

示例 Example:
  ./app -port 18080 -dir /path/to/files -title "我的资源站"
`)
		return
	}

	// 读取并解析内嵌的模板文件
	tplContent, err := embeddedFiles.ReadFile("index.html")
	if err != nil {
		log.Fatalf("读取模板失败: %v", err)
	}
	tpl, err := template.New("index").Parse(string(tplContent))
	if err != nil {
		log.Fatalf("解析模板失败: %v", err)
	}

	// 获取当前可执行文件的名字，用于隐藏自身
	exePath, _ := os.Executable()
	exeBase := filepath.Base(exePath)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(*dir, filepath.Clean(r.URL.Path))

		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "文件或目录不存在", http.StatusNotFound)
				log.Printf("文件不存在: %s, 错误: %v", path, err)
			} else if os.IsPermission(err) {
				http.Error(w, "没有访问权限", http.StatusForbidden)
				log.Printf("权限不足: %s, 错误: %v", path, err)
			} else {
				http.Error(w, "无法访问文件", http.StatusInternalServerError)
				log.Printf("访问文件失败: %s, 错误: %v", path, err)
			}
			return
		}

		if info.IsDir() {
			files, err := os.ReadDir(path)
			if err != nil {
				http.Error(w, "无法读取目录", http.StatusInternalServerError)
				log.Printf("读取目录失败: %s, 错误: %v", path, err)
				return
			}

			var filtered []os.DirEntry
			for _, f := range files {
				if strings.HasPrefix(f.Name(), ".") || f.Name() == exeBase {
					continue
				}
				filtered = append(filtered, f)
			}

			sort.Slice(filtered, func(i, j int) bool {
				iInfo, _ := filtered[i].Info()
				jInfo, _ := filtered[j].Info()
				if iInfo.IsDir() && !jInfo.IsDir() {
					return true
				}
				if !iInfo.IsDir() && jInfo.IsDir() {
					return false
				}
				return strings.ToLower(filtered[i].Name()) < strings.ToLower(filtered[j].Name())
			})

			var fileItems []FileItem
			for _, f := range filtered {
				fi, _ := f.Info()
				size := "-"
				if !fi.IsDir() {
					size = humanReadableSize(fi.Size())
				}

				fileItems = append(fileItems, FileItem{
					Name:       f.Name(),
					Href:       filepath.ToSlash(filepath.Join(r.URL.Path, f.Name())),
					Size:       size,
					IsDir:      fi.IsDir(),
					IconSVG:    getFileIcon(f.Name(), fi.IsDir()),
					ModTime:    formatModTime(fi.ModTime()),
					Type:       getFileType(f.Name()),
					CanPreview: !fi.IsDir() && canPreview(f.Name()),
				})
			}

			err = tpl.Execute(w, PageData{
				Title:       *title,
				Files:       fileItems,
				CurrentPath: r.URL.Path,
			})
			if err != nil {
				log.Println("模板执行错误:", err)
				http.Error(w, "页面渲染失败", http.StatusInternalServerError)
			}
		} else {
			query := r.URL.Query()
			if query.Get("preview") == "1" {
				ext := strings.ToLower(filepath.Ext(path))
				if imageExts[ext] {
					w.Header().Set("Content-Type", "image/"+strings.TrimPrefix(ext, "."))
					http.ServeFile(w, r, path)
				} else if textExts[ext] || codeExts[ext] {
					content, err := os.ReadFile(path)
					if err != nil {
						http.Error(w, "无法读取文件内容", http.StatusInternalServerError)
						log.Printf("读取文件失败: %s, 错误: %v", path, err)
						return
					}
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					w.Write(content)
				} else {
					http.Error(w, "不支持预览此文件类型", http.StatusBadRequest)
				}
			} else if query.Get("download") == "1" {
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(path)))
				http.ServeFile(w, r, path)
			} else {
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(path)))
				http.ServeFile(w, r, path)
			}
		}
	})

	// 监听端口，自动端口递增重试
	currentPort := *port
	for {
		addr := fmt.Sprintf(":%d", currentPort)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("端口 %d 被占用，尝试 %d...", currentPort, currentPort+1)
			currentPort++
			continue
		}
		log.Printf("服务器启动成功: http://localhost:%d", currentPort)
		log.Fatal(http.Serve(ln, nil))
	}
}
