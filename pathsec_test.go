package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCleanURLPath(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "/", want: ""},
		{in: "", want: ""},
		{in: "/a", want: "a"},
		{in: "/a/b.txt", want: "a/b.txt"},
		{in: "/a%20b/c.txt", want: "a b/c.txt"},
		{in: "/中文/目录/文件.txt", want: "中文/目录/文件.txt"},
		{in: "/a//b", want: "a/b"},   // 连续斜杠折叠
		{in: "/a/./b", want: "a/b"},  // 点段忽略
		{in: "/./x", want: "x"},      // 顶部点段忽略
		{in: "/a/b/", want: "a/b"},   // 尾部斜杠
		{in: "/..", wantErr: true},   // 非法：向上穿越
		{in: "/../x", wantErr: true}, // 非法
		{in: "/a/../../x", wantErr: true},
		{in: "/a/..", wantErr: true},
		{in: "/%2e%2e/x", wantErr: true},      // 编码的 ..
		{in: "/a/%2E%2E/b", wantErr: true},    // 编码的 ..
		{in: "/%2fetc/passwd", wantErr: true}, // 段内含 / 的编码
		{in: "/a%2fb", wantErr: true},         // 段内含 / 的编码
		{in: "/a%00b", wantErr: true},         // NUL
		{in: "/a%zz", wantErr: true},          // 非法转义
	}
	for _, tt := range tests {
		got, err := cleanURLPath(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("cleanURLPath(%q) 期望报错，却返回 %q", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("cleanURLPath(%q) 意外报错: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("cleanURLPath(%q) = %q, 期望 %q", tt.in, got, tt.want)
		}
	}
}

func TestCleanURLPathBackslashOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅 Windows 检查反斜杠分隔符")
	}
	if _, err := cleanURLPath("/..\\..\\windows"); err == nil {
		t.Error("Windows 下应拒绝含反斜杠的路径")
	}
}

func TestFullPathWithinRoot(t *testing.T) {
	root := t.TempDir()
	srv, err := NewServer(Config{Dir: root, Title: "t", Log: discardLogger()})
	if err != nil {
		t.Fatal(err)
	}

	// 正常路径
	for _, rel := range []string{"", "a", "a/b", "文件/子"} {
		full, err := srv.fullPath(rel)
		if err != nil {
			t.Fatalf("fullPath(%q) 意外报错: %v", rel, err)
		}
		within, _ := isWithin(srv.rootAbs, full)
		if !within {
			t.Errorf("fullPath(%q)=%q 逃出根目录", rel, full)
		}
	}

	// 词法逃逸
	for _, rel := range []string{"..", "../x", "a/../../x"} {
		if _, err := srv.fullPath(rel); err == nil {
			t.Errorf("fullPath(%q) 期望越界报错", rel)
		}
	}

	// 根目录自身
	full, err := srv.fullPath("")
	if err != nil {
		t.Fatalf("fullPath(root) 报错: %v", err)
	}
	if filepath.Clean(full) != srv.rootAbs {
		t.Errorf("root fullPath = %q, want %q", full, srv.rootAbs)
	}
}

func TestSymlinkEscapeRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 上创建符号链接通常需要管理员权限")
	}
	base := t.TempDir()
	root := filepath.Join(base, "share")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	// 根目录之外创建秘密文件
	secret := filepath.Join(base, "secret.txt")
	if err := os.WriteFile(secret, []byte("top-secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 在共享目录内创建指向外部的符号链接
	if err := os.Symlink(secret, filepath.Join(root, "leak.txt")); err != nil {
		t.Skip("无法创建符号链接:", err)
	}

	srv, err := NewServer(Config{Dir: root, Title: "t", Log: discardLogger()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.fullPath("leak.txt"); err == nil {
		t.Error("指向外部的符号链接应被拒绝")
	}
}

func discardLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}
