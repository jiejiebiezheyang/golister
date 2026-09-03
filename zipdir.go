package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// maxZipBytes 限制单次打包的总大小上限（4 GiB），
// 避免误触超大目录把服务端流量拖垮；命中时提前终止并返回提示。
const maxZipBytes = int64(4) << 30

// streamZipDir 把 rel 指向的目录流式打包为 zip 写入 w。
// 根目录自身以 <目录名>/ 作为归档顶层文件夹；隐藏规则与列表一致。
func (s *Server) streamZipDir(w http.ResponseWriter, rel string, hide func(string) bool) error {
	full, err := s.fullPath(rel)
	if err != nil {
		return err
	}
	baseName := zipBaseName(rel)
	if rel == "" {
		baseName = filepath.Base(filepath.Clean(s.rootAbs))
	}

	zw := zip.NewWriter(w)
	closeOnce := func() error { return zw.Close() }

	var total int64
	var walkErr error

	addDir := func(archiveDir string, d fs.DirEntry) error {
		info, err := d.Info()
		if err != nil {
			return nil
		}
		hdr := &zip.FileHeader{
			Name:     archiveDir + "/",
			Method:   zip.Store,
			Modified: info.ModTime(),
		}
		hdr.SetMode(fs.ModeDir | 0o755)
		_, err = zw.CreateHeader(hdr)
		return err
	}

	addFile := func(archiveName, srcPath string, fi fs.FileInfo) error {
		f, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		defer f.Close()

		hdr := &zip.FileHeader{
			Name:     archiveName,
			Method:   zip.Deflate,
			Modified: fi.ModTime(),
		}
		hdr.SetMode(fi.Mode())
		entry, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		n, err := io.Copy(entry, f)
		if err != nil {
			return err
		}
		total += n
		return nil
	}

	walkErr = filepath.WalkDir(full, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// 无权限/删除的条目：跳过，不中断整体打包
			return nil
		}
		if total >= maxZipBytes {
			return fmt.Errorf("目录过大，超过打包上限")
		}
		// 符号链接一律不入包（其目标可能指向共享目录之外）
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		name := d.Name()
		if p != full && hide != nil && hide(name) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if p == full {
			return nil // 顶层目录本身不重复入包（由其子项体现）
		}

		relPart, err := filepath.Rel(full, p)
		if err != nil {
			return err
		}
		relPart = filepath.ToSlash(relPart)
		archiveName := baseName + "/" + relPart

		if d.IsDir() {
			return addDir(archiveName, d)
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		return addFile(archiveName, p, fi)
	})
	if walkErr != nil {
		_ = closeOnce()
		return walkErr
	}
	return closeOnce()
}

// zipBaseName 返回用作下载文件名的目录归档名（仅命名用）。
func zipBaseName(rel string) string {
	if rel == "" {
		return "root"
	}
	parts := strings.Split(rel, "/")
	return parts[len(parts)-1]
}
