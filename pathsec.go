package main

import (
	"errors"
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	errBadPath = errors.New("无效的请求路径")
	errOutside = errors.New("路径超出共享目录范围")
)

// cleanURLPath 把 URL 的转义路径解码成一个以 "/" 分隔的干净相对路径
// （不带前导斜杠；根目录返回 ""）。
//
// 安全要点：
//   - 逐段解码，拒绝 ".." 与 "." 段（含编码形式），拒绝 NUL；
//   - 拒绝段内残留的 "/"（即 %2F，防双重解码混淆）与反斜杠 "\"
//     （Windows 上反斜杠也是目录分隔符，可能被用作穿越）；
//   - 不依赖 filepath.Clean，纯以 "/" 语义处理，跨平台一致。
func cleanURLPath(escaped string) (string, error) {
	if escaped == "" {
		return "", nil
	}
	escaped = strings.TrimPrefix(escaped, "/")
	if escaped == "" {
		return "", nil
	}
	rawSegs := strings.Split(escaped, "/")
	segs := make([]string, 0, len(rawSegs))
	for _, raw := range rawSegs {
		if raw == "" {
			continue // 折叠连续斜杠
		}
		dec, err := url.PathUnescape(raw)
		if err != nil {
			return "", errBadPath
		}
		if strings.ContainsRune(dec, 0) {
			return "", errBadPath
		}
		if dec == "." {
			continue // 点段无害，跳过（等价 path.Clean 行为）
		}
		if dec == ".." || strings.Contains(dec, "/") ||
			(runtime.GOOS == "windows" && strings.Contains(dec, `\`)) {
			return "", errBadPath
		}
		segs = append(segs, dec)
	}
	return path.Join(segs...), nil
}

// fullPath 把清理后的相对路径拼接到共享根目录下，并做两次防逃逸校验：
//  1. 词法包含校验（filepath.Rel 不得向上逃出根目录）；
//  2. 真实路径校验（解析符号链接后仍须位于根目录真实路径内）。
//
// 返回拼接后的完整路径；文件不存在时也返回该路径，由调用方 stat。
func (s *Server) fullPath(rel string) (string, error) {
	full := filepath.Join(s.rootAbs, filepath.FromSlash(rel))

	// 词法包含校验（处理 Windows 卷/分隔符边界情况）
	if ok, err := isWithin(s.rootAbs, full); err != nil || !ok {
		return "", errOutside
	}

	// 真实路径校验（防符号链接逃逸）
	realFull, err := filepath.EvalSymlinks(full)
	if err != nil {
		// 文件不存在/中间组件缺失：交由后续 os.Stat 处理
		return full, nil
	}
	if ok, err := isWithin(s.rootReal, realFull); err != nil || !ok {
		return "", errOutside
	}
	return full, nil
}

// isWithin 报告 p 是否位于 root 内部（root 自身也算在内）。
func isWithin(root, p string) (bool, error) {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false, err
	}
	if rel == "." {
		return true, nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, nil
	}
	return true, nil
}

// escapeSeg 转义单个路径段用于构造链接。
func escapeSeg(seg string) string {
	return url.PathEscape(seg)
}

// dirHrefBase 返回当前相对路径对应 URL 的目录基址（以 "/" 结尾，
// 例如 rel="a b" -> "/a%20b/"，根目录 -> "/"）。
func dirHrefBase(rel string) string {
	if rel == "" {
		return "/"
	}
	segs := strings.Split(rel, "/")
	for i := range segs {
		segs[i] = escapeSeg(segs[i])
	}
	return "/" + strings.Join(segs, "/") + "/"
}
