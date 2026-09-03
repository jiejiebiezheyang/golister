package main

import (
	"mime"
	"path/filepath"
	"strings"
)

// 文件分类 key：全部与模板内的图标 id（#i-<kind>）一一对应，
// Go 代码不再输出任何 HTML/SVG 字符串。
const (
	kindDir     = "dir"
	kindImage   = "image"
	kindText    = "text"
	kindPDF     = "pdf"
	kindArchive = "archive"
	kindFile    = "file"
)

// 图片扩展名（可内联预览）
var imageExts = stringSet(
	".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico", ".avif",
)

// 文本/代码扩展名（可文本预览）
var textExts = stringSet(
	".txt", ".md", ".markdown", ".json", ".xml", ".html", ".htm", ".css",
	".js", ".mjs", ".ts", ".tsx", ".jsx", ".go", ".py", ".java", ".c", ".cpp",
	".cc", ".cxx", ".h", ".hpp", ".sh", ".bash", ".zsh", ".bat", ".cmd",
	".ps1", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".conf", ".log",
	".sql", ".rb", ".php", ".rs", ".swift", ".kt", ".scala", ".lua", ".r",
	".dockerfile", ".gitignore", ".env", ".properties", ".csv", ".tsv",
	".vue", ".svelte", ".cs", ".pl", ".ex", ".exs", ".erl", ".hrl",
	".jl", ".dart", ".groovy", ".gradle", ".tf", ".proto", ".graphql",
	".coffee", ".nim", ".zig", ".v", ".fs", ".fsx",
)

var pdfExts = stringSet(".pdf")

var archiveExts = stringSet(
	".zip", ".rar", ".7z", ".tar", ".gz", ".tgz", ".bz2", ".xz", ".zst", ".tar.gz",
)

func stringSet(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}

// classify 根据文件名与是否为目录返回分类 key。
func classify(name string, isDir bool) string {
	if isDir {
		return kindDir
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch {
	case imageExts[ext]:
		return kindImage
	case textExts[ext]:
		return kindText
	case pdfExts[ext]:
		return kindPDF
	case archiveExts[ext]:
		return kindArchive
	default:
		return kindFile
	}
}

// kindLabel 返回分类的中文展示名。
func kindLabel(kind string) string {
	switch kind {
	case kindDir:
		return "目录"
	case kindImage:
		return "图片"
	case kindText:
		return "文本"
	case kindPDF:
		return "PDF"
	case kindArchive:
		return "压缩包"
	default:
		return "文件"
	}
}

// canPreview 表示该分类可被内联预览（图片或文本）。
func canPreview(kind string) bool {
	return kind == kindImage || kind == kindText
}

// canZip 表示该分类可被打包下载（仅目录）。
func canZip(kind string) bool {
	return kind == kindDir
}

// mimeFor 根据扩展名返回 Content-Type。
func mimeFor(name string) string {
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(name))); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
