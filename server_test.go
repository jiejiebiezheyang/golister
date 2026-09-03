package main

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, root string, pageSize int) *Server {
	t.Helper()
	srv, err := NewServer(Config{Dir: root, Title: "测试站", PageSize: pageSize, Log: discardLogger()})
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func doGet(t *testing.T, srv *Server, url string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestServerRootListing(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "hello.txt"), "hi")
	mustWrite(t, filepath.Join(root, ".secret"), "s")
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	srv := newTestServer(t, root, 200)

	rec := doGet(t, srv, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "hello.txt") {
		t.Error("页面应包含 hello.txt")
	}
	if strings.Contains(body, ".secret") {
		t.Error("页面不应泄露隐藏文件")
	}
	if !strings.Contains(body, "docs") {
		t.Error("页面应包含 docs 目录")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestServerSubdir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "a", "b", "x.txt"), "x")
	srv := newTestServer(t, root, 200)

	rec := doGet(t, srv, "/a/b/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /a/b/ = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "x.txt") {
		t.Error("子目录页面应包含 x.txt")
	}
}

func TestServerDirRedirect(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	srv := newTestServer(t, root, 200)
	rec := doGet(t, srv, "/sub")
	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("GET /sub = %d, want 301", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/sub/" {
		t.Errorf("Location = %q, want /sub/", loc)
	}
}

func TestServerDownload(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "数据 文件.txt"), "文件内容")
	srv := newTestServer(t, root, 200)

	rec := doGet(t, srv, "/%E6%95%B0%E6%8D%AE%20%E6%96%87%E4%BB%B6.txt")
	if rec.Code != http.StatusOK {
		t.Fatalf("download = %d", rec.Code)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition = %q", cd)
	}
	if rec.Body.String() != "文件内容" {
		t.Errorf("下载内容不符: %q", rec.Body.String())
	}
}

func TestServerTextPreview(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "note.txt"), "预览文本")
	srv := newTestServer(t, root, 200)

	rec := doGet(t, srv, "/note.txt?preview=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("preview = %d", rec.Code)
	}
	if rec.Body.String() != "预览文本" {
		t.Errorf("预览内容不符: %q", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q", ct)
	}
	if rec.Header().Get("X-Content-Truncated") != "" {
		t.Error("小文件不应标记截断")
	}
}

func TestServerTextPreviewTruncated(t *testing.T) {
	root := t.TempDir()
	big := strings.Repeat("x", maxPreviewBytes+4096)
	mustWrite(t, filepath.Join(root, "big.txt"), big)
	srv := newTestServer(t, root, 200)

	rec := doGet(t, srv, "/big.txt?preview=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("preview = %d", rec.Code)
	}
	if rec.Header().Get("X-Content-Truncated") != "1" {
		t.Error("超大文件应标记 X-Content-Truncated")
	}
	if rec.Body.Len() > maxPreviewBytes+2048 {
		t.Errorf("预览体量应被限制，实际 %d", rec.Body.Len())
	}
}

func TestServerImagePreview(t *testing.T) {
	root := t.TempDir()
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(filepath.Join(root, "pic.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newTestServer(t, root, 200)

	rec := doGet(t, srv, "/pic.png?preview=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("image preview = %d", rec.Code)
	}
	// 内联预览不应带附件头
	if cd := rec.Header().Get("Content-Disposition"); cd != "" {
		t.Errorf("图片预览不应带 Content-Disposition: %q", cd)
	}
	if ct := rec.Header().Get("Content-Type"); ct != mimeFor("pic.png") {
		t.Errorf("Content-Type = %q, want %q", ct, mimeFor("pic.png"))
	}
}

func TestServerZip(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "docs", "a.txt"), "AAA")
	mustWrite(t, filepath.Join(root, "docs", "nested", "b.txt"), "BBB")
	mustWrite(t, filepath.Join(root, "docs", ".ignore"), "no")
	srv := newTestServer(t, root, 200)

	rec := doGet(t, srv, "/docs/?zip=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("zip = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, ".zip") {
		t.Errorf("Content-Disposition = %q", cd)
	}

	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("响应不是合法 zip: %v", err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	joined := strings.Join(names, "|")
	for _, want := range []string{"docs/a.txt", "docs/nested/b.txt"} {
		if !strings.Contains(joined, want) {
			t.Errorf("zip 缺少 %s，实际: %s", want, joined)
		}
	}
	if strings.Contains(joined, ".ignore") {
		t.Error("zip 不应包含隐藏文件")
	}
}

func TestServerNotFound(t *testing.T) {
	root := t.TempDir()
	srv := newTestServer(t, root, 200)
	rec := doGet(t, srv, "/no-such.txt")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET missing = %d, want 404", rec.Code)
	}
}

func TestServerTraversalBlocked(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "ok.txt"), "ok")
	srv := newTestServer(t, root, 200)

	// 各种穿越尝试都不应返回 200 且不能带出内容
	urls := []string{
		"/..%2f..%2fetc%2fpasswd",
		"/..%5c..%5cetc%5cpasswd",
		"/.%2e/%2e%2e/secret",
		"/a/../../b",
	}
	for _, u := range urls {
		rec := doGet(t, srv, u)
		if rec.Code == http.StatusOK {
			t.Errorf("穿越路径 %s 返回了 200", u)
		}
	}
}

func TestServerPagination(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		mustWrite(t, filepath.Join(root, "f"+string(rune('a'+i))+".txt"), "x")
	}
	srv := newTestServer(t, root, 3) // 每页 3 条

	rec1 := doGet(t, srv, "/")
	if !strings.Contains(rec1.Body.String(), "fa.txt") {
		t.Error("第 1 页应包含 fa.txt")
	}
	if strings.Contains(rec1.Body.String(), "fe.txt") {
		t.Error("第 1 页不应包含 fe.txt（应分页）")
	}
	if !strings.Contains(rec1.Body.String(), "1 / 2") {
		t.Error("应显示分页 1 / 2")
	}

	rec2 := doGet(t, srv, "/?page=2")
	if !strings.Contains(rec2.Body.String(), "fe.txt") {
		t.Error("第 2 页应包含 fe.txt")
	}
	if strings.Contains(rec2.Body.String(), "fa.txt") {
		t.Error("第 2 页不应包含 fa.txt")
	}
}

func TestServerQueryFilter(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "alpha.txt"), "a")
	mustWrite(t, filepath.Join(root, "beta.txt"), "b")
	srv := newTestServer(t, root, 200)

	rec := doGet(t, srv, "/?q=beta")
	if !strings.Contains(rec.Body.String(), "beta.txt") {
		t.Error("搜索 beta 应包含 beta.txt")
	}
	if strings.Contains(rec.Body.String(), "alpha.txt") {
		t.Error("搜索 beta 不应包含 alpha.txt")
	}
}

func TestServerMethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	srv := newTestServer(t, root, 200)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST / = %d, want 405", rec.Code)
	}
}
