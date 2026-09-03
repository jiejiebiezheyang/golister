package main

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

// maxPreviewBytes 文本预览单次读取上限：超过则截断并提示（避免大文件占满内存）。
const maxPreviewBytes = 2 << 20 // 2 MiB

// handleFile 处理单个文件的请求：
//   - ?preview=1 且为图片：内联展示原图；
//   - ?preview=1 且为文本：以纯文本内联预览（截断超大文件）；
//   - 其余情况：作为附件下载。
func (s *Server) handleFile(w http.ResponseWriter, r *http.Request, rel, full string, info os.FileInfo) {
	q := r.URL.Query()
	if q.Get("preview") == "1" && canPreview(classify(info.Name(), false)) {
		kind := classify(info.Name(), false)
		if kind == kindImage {
			s.serveImageInline(w, r, full, info)
		} else {
			s.serveTextPreview(w, r, full, info)
		}
		return
	}
	// 默认按附件下载
	s.serveDownload(w, r, full, info)
}

// serveDownload 附件下载（支持 Range/HEAD/断点续传，由 ServeContent 处理）。
func (s *Server) serveDownload(w http.ResponseWriter, r *http.Request, full string, info os.FileInfo) {
	f, err := os.Open(full)
	if err != nil {
		http.Error(w, "无法打开文件", http.StatusInternalServerError)
		s.log.Printf("打开文件失败 %s: %v", full, err)
		return
	}
	defer f.Close()

	name := filepath.Base(full)
	w.Header().Set("Content-Type", mimeFor(name))
	setContentDisposition(w, name)
	http.ServeContent(w, r, name, info.ModTime(), f)
}

// serveImageInline 图片内联预览。
func (s *Server) serveImageInline(w http.ResponseWriter, r *http.Request, full string, info os.FileInfo) {
	f, err := os.Open(full)
	if err != nil {
		http.Error(w, "无法打开文件", http.StatusInternalServerError)
		s.log.Printf("打开图片失败 %s: %v", full, err)
		return
	}
	defer f.Close()

	name := filepath.Base(full)
	w.Header().Set("Content-Type", mimeFor(name))
	http.ServeContent(w, r, name, info.ModTime(), f)
}

// serveTextPreview 文本预览：超大文件只取前 maxPreviewBytes 字节，
// 并在响应头标记 X-Content-Truncated，由前端提示。
func (s *Server) serveTextPreview(w http.ResponseWriter, r *http.Request, full string, info os.FileInfo) {
	f, err := os.Open(full)
	if err != nil {
		http.Error(w, "无法打开文件", http.StatusInternalServerError)
		s.log.Printf("打开文本失败 %s: %v", full, err)
		return
	}
	defer f.Close()

	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// 小文件直接全量返回；大文件截断
	truncated := info.Size() > maxPreviewBytes
	reader := io.LimitReader(f, maxPreviewBytes+1)
	buf := make([]byte, 0, maxPreviewBytes)
	tmp := make([]byte, 32*1024)
	for {
		n, rerr := reader.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) > maxPreviewBytes {
				buf = buf[:maxPreviewBytes]
				truncated = true
				break
			}
		}
		if rerr != nil {
			break
		}
	}

	if truncated {
		w.Header().Set("X-Content-Truncated", "1")
	}
	if _, err := w.Write(buf); err != nil {
		s.log.Printf("写出预览失败 %s: %v", full, err)
	}
}

// formatAttachment 生成 RFC 5987 安全的 Content-Disposition 值，
// 非 ASCII 文件名会自动转成 filename*=UTF-8”...（修复原始字符串拼接的安全问题）。
func formatAttachment(filename string) string {
	// mime.FormatMediaType 会正确给引号/非 ASCII 编码
	//（ASCII 文件名走 filename="..."，其他走 filename*= 形式）
	mediaType := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	if mediaType == "" {
		return "attachment"
	}
	return mediaType
}

// setContentDisposition 设置附件下载响应头。
func setContentDisposition(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Disposition", formatAttachment(filename))
}
