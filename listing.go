package main

import (
	"os"
	"sort"
	"strings"
	"time"
)

// fileEntry 是目录列举的原始数据（大小/时间保留原始数值，供排序与格式化）。
type fileEntry struct {
	name    string
	rel     string // 相对共享根的 "/" 路径（含自身），用于 zip/链接
	isDir   bool
	size    int64
	modTime time.Time
	kind    string
}

// listDir 列举共享目录下 rel 对应的目录（rel 为 "/" 分隔的相对路径，根为 ""），
// 返回排序后（目录优先）且支持搜索过滤的条目列表。
//
//   - hide：返回 true 表示隐藏（如以 "." 开头、可执行文件自身）；
//   - query：非空时按名称做不区分大小写的子串过滤。
func (s *Server) listDir(rel string, hide func(string) bool, query string) ([]fileEntry, error) {
	full, err := s.fullPath(rel)
	if err != nil {
		return nil, err
	}
	des, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}

	entries := make([]fileEntry, 0, len(des))
	for _, de := range des {
		name := de.Name()
		if hide != nil && hide(name) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
			continue
		}
		info, err := de.Info()
		if err != nil {
			// 条目在读取间隙消失/无权限：跳过而非 panic（修复原实现的潜在 nil 解引用）
			continue
		}
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		entries = append(entries, fileEntry{
			name:    name,
			rel:     childRel,
			isDir:   info.IsDir(),
			size:    info.Size(),
			modTime: info.ModTime(),
			kind:    classify(name, info.IsDir()),
		})
	}
	return entries, nil
}

// sortEntries 排序：目录永远在最前（目录内部按名称不区分大小写升序），
// 其余条目按所选键（name/size/mtime）与方向排序，键相同按名称兜底。
func sortEntries(entries []fileEntry, sortKey, order string) {
	asc := order != "desc"
	less := func(a, b fileEntry) bool {
		if a.isDir != b.isDir {
			return a.isDir // 目录优先
		}
		if a.isDir {
			// 目录之间：固定按名称升序，避免视觉跳跃
			return strings.ToLower(a.name) < strings.ToLower(b.name)
		}
		cmp := 0
		switch sortKey {
		case "size":
			cmp = compareInt64(a.size, b.size)
		case "mtime":
			if !a.modTime.Equal(b.modTime) {
				if a.modTime.Before(b.modTime) {
					cmp = -1
				} else {
					cmp = 1
				}
			}
		default: // name
			cmp = strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
		}
		if cmp == 0 {
			cmp = strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
		}
		if !asc {
			cmp = -cmp
		}
		return cmp < 0
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return less(entries[i], entries[j])
	})
}

func compareInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
