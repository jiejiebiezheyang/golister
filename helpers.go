package main

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// defaultPageSize 默认每页条目数。
const defaultPageSize = 200

// humanReadableSize 人类可读大小。
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

// formatModTime 本地时区的展示格式。
func formatModTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05")
}

// parsePage 解析 page 参数，非法/缺省返回 1。
func parsePage(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// validSortKey 校验排序键，非法回退到 name。
func validSortKey(raw string) string {
	switch raw {
	case "name", "size", "mtime":
		return raw
	default:
		return "name"
	}
}

// validOrder 校验排序方向，非 desc 一律升序。
func validOrder(raw string) string {
	if raw == "desc" {
		return "desc"
	}
	return "asc"
}

// listingQuery 构造目录页的查询串（sort/order/q/page）。
func listingQuery(sortKey, order, q string, page int) string {
	v := url.Values{}
	v.Set("sort", sortKey)
	v.Set("order", order)
	if q != "" {
		v.Set("q", q)
	}
	v.Set("page", strconv.Itoa(page))
	return v.Encode()
}
