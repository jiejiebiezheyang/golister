package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func timeOld() time.Time { return time.Unix(1000, 0) }
func timeNew() time.Time { return time.Unix(2000, 0) }

// setupTree 建一个测试目录树。
//   - root/a.txt（空文件）
//   - root/zdir/（空目录）
//   - root/.hidden（隐藏文件）
//   - root/big.bin（3 字节）
//   - root/mdir/ 内含 2.txt
func setupTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.txt"), "")
	mustWrite(t, filepath.Join(root, "big.bin"), "abc")
	mustWrite(t, filepath.Join(root, ".hidden"), "h")
	mustWrite(t, filepath.Join(root, "zzz"), "z")
	if err := os.MkdirAll(filepath.Join(root, "zdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "mdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "mdir", "2.txt"), "22")
	return root
}

func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListDirHidesAndSorts(t *testing.T) {
	root := setupTree(t)
	srv, err := NewServer(Config{Dir: root, Title: "t", Log: discardLogger()})
	if err != nil {
		t.Fatal(err)
	}

	hide := func(name string) bool { return srv.hide(name) }
	entries, err := srv.listDir("", hide, "")
	if err != nil {
		t.Fatal(err)
	}
	sortEntries(entries, "name", "asc") // 排序发生在 handler 中，此处显式调用
	// 隐藏文件不应出现
	for _, e := range entries {
		if e.name == ".hidden" {
			t.Error("列表不应包含隐藏文件 .hidden")
		}
	}
	// 目录应排在最前，且目录内按名称升序
	if len(entries) < 2 {
		t.Fatalf("条目太少: %d", len(entries))
	}
	if !entries[0].isDir || entries[0].name != "mdir" {
		t.Errorf("期望首位目录 mdir，got %+v", entries[0])
	}
	if !entries[1].isDir || entries[1].name != "zdir" {
		t.Errorf("期望第二位目录 zdir，got %+v", entries[1])
	}
	// 之后是文件按名称
	if entries[2].isDir || entries[2].name != "a.txt" {
		t.Errorf("期望文件 a.txt，got %+v", entries[2])
	}
}

func TestListDirQuery(t *testing.T) {
	root := setupTree(t)
	srv, _ := NewServer(Config{Dir: root, Title: "t", Log: discardLogger()})
	entries, err := srv.listDir("", nil, "mdir")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].name != "mdir" {
		t.Errorf("q=mdir 应只匹配 mdir，got %+v", entries)
	}
	entries, err = srv.listDir("", nil, "NOPE")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("q=NOPE 应无结果，got %d", len(entries))
	}
}

func TestListDirNested(t *testing.T) {
	root := setupTree(t)
	srv, _ := NewServer(Config{Dir: root, Title: "t", Log: discardLogger()})
	entries, err := srv.listDir("mdir", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].name != "2.txt" || entries[0].rel != "mdir/2.txt" {
		t.Errorf("嵌套目录列举异常: %+v", entries)
	}
}

func TestListDirMissing(t *testing.T) {
	root := setupTree(t)
	srv, _ := NewServer(Config{Dir: root, Title: "t", Log: discardLogger()})
	if _, err := srv.listDir("no-such-dir", nil, ""); err == nil {
		t.Error("不存在的目录应报错")
	}
}

func TestSortEntries(t *testing.T) {
	es := []fileEntry{
		{name: "b.txt", isDir: false, size: 30},
		{name: "a.txt", isDir: false, size: 10},
		{name: "C.txt", isDir: false, size: 20},
		{name: "Adir", isDir: true},
		{name: "bdir", isDir: true},
	}
	sortEntries(es, "name", "asc")
	// 目录在前，目录之间按名称升序
	if !es[0].isDir || es[0].name != "Adir" {
		t.Errorf("目录应排最前且按名升序: %+v", es)
	}
	if !es[1].isDir || es[1].name != "bdir" {
		t.Errorf("目录第二位应为 bdir: %+v", es)
	}
	// 文件名称不区分大小写排序
	if es[2].isDir || es[2].name != "a.txt" {
		t.Errorf("名称应不区分大小写排序: %+v", es)
	}
	if es[3].name != "b.txt" || es[4].name != "C.txt" {
		t.Errorf("name asc 顺序错误: %+v", es)
	}

	es = []fileEntry{
		{name: "b.txt", isDir: false, size: 30},
		{name: "a.txt", isDir: false, size: 10},
		{name: "C.txt", isDir: false, size: 20},
		{name: "Adir", isDir: true},
	}
	sortEntries(es, "size", "asc")
	// 文件在目录之后，按大小升序
	if es[0].name != "Adir" {
		t.Errorf("目录应始终在顶部: %+v", es)
	}
	if es[1].name != "a.txt" || es[2].name != "C.txt" || es[3].name != "b.txt" {
		t.Errorf("size 升序错误: %+v", es)
	}

	sortEntries(es, "size", "desc")
	if es[1].name != "b.txt" {
		t.Errorf("size 降序文件首位应为 b.txt: %+v", es)
	}

	// mtime 排序
	es = []fileEntry{
		{name: "old", modTime: timeOld()},
		{name: "new", modTime: timeNew()},
		{name: "Adir", isDir: true},
	}
	sortEntries(es, "mtime", "asc")
	if es[1].name != "old" {
		t.Errorf("mtime asc 首位文件应为 old: %+v", es)
	}
	sortEntries(es, "mtime", "desc")
	if es[1].name != "new" {
		t.Errorf("mtime desc 首位文件应为 new: %+v", es)
	}
}
