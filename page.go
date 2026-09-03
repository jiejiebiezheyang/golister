package main

import (
	"net/url"
	"strings"
)

// crumb 面包屑中的一段。
type crumb struct {
	Label string
	URL   string
	Last  bool
}

// fileRow 列表中的一行（展示层数据，不含 HTML 字符串）。
type fileRow struct {
	Name        string
	Href        string
	Size        string
	ModTime     string
	Kind        string // dir/image/text/pdf/archive/file
	KindLabel   string
	IsDir       bool
	CanPreview  bool
	PreviewHref string
	ZipHref     string
}

// pageInfo 分页信息。
type pageInfo struct {
	Page      int
	PageCount int
	Total     int
	PerPage   int
	HasPrev   bool
	HasNext   bool
	PrevURL   string
	NextURL   string
}

// pageData 模板数据。
type pageData struct {
	Title        string
	CurrentPath  string // 展示用当前路径（以 / 开头）
	Rel          string // 相对根路径（"" 表示根）
	Crumbs       []crumb
	ShowParent   bool
	ParentHref   string
	Files        []fileRow
	Empty        bool
	Q            string
	Sort         string
	Order        string
	SortURLs     map[string]string // 点击表头排序的链接
	SearchAction string
	Page         pageInfo
	ZipHref      string
	HasQuery     bool
	TotalLabel   string
}

// pageDataArgs 渲染目录页所需的原始参数。
type pageDataArgs struct {
	page      int
	pageCount int
	total     int
	sort      string
	order     string
	search    string
}

// buildPageData 组装目录页模板数据。
func (s *Server) buildPageData(rel string, entries []fileEntry, a pageDataArgs) pageData {
	base := dirHrefBase(rel) // 以 / 结尾的目录 URL

	// 面包屑
	var crumbs []crumb
	crumbs = append(crumbs, crumb{Label: "Index", URL: "/", Last: rel == ""})
	if rel != "" {
		segHref := ""
		segs := strings.Split(rel, "/")
		for i, seg := range segs {
			segHref += "/" + escapeSeg(seg)
			crumbs = append(crumbs, crumb{
				Label: seg,
				URL:   segHref + "/",
				Last:  i == len(segs)-1,
			})
		}
	}

	// 排序链接
	mkSort := func(key, order string) string {
		v := url.Values{}
		v.Set("sort", key)
		v.Set("order", order)
		if a.search != "" {
			v.Set("q", a.search)
		}
		v.Set("page", "1")
		return base + "?" + v.Encode()
	}
	sortURLs := map[string]string{
		"name":  mkSort("name", nextOrder(a.sort, "name", a.order)),
		"size":  mkSort("size", nextOrder(a.sort, "size", a.order)),
		"mtime": mkSort("mtime", nextOrder(a.sort, "mtime", a.order)),
	}

	// 父目录
	parentHref := ""
	showParent := rel != ""
	if showParent {
		parts := strings.Split(rel, "/")
		parentRel := strings.Join(parts[:len(parts)-1], "/")
		parentHref = dirHrefBase(parentRel)
	}

	// 行数据
	rows := make([]fileRow, 0, len(entries))
	for _, e := range entries {
		href := base + escapeSeg(e.name)
		row := fileRow{
			Name:      e.name,
			Href:      href,
			Kind:      e.kind,
			KindLabel: kindLabel(e.kind),
			IsDir:     e.isDir,
		}
		if e.isDir {
			row.Size = "-"
			row.Href = href + "/"
			row.ZipHref = href + "/?zip=1"
		} else {
			row.Size = humanReadableSize(e.size)
			row.CanPreview = canPreview(e.kind)
			if row.CanPreview {
				row.PreviewHref = href + "?preview=1"
			}
		}
		row.ModTime = formatModTime(e.modTime)
		rows = append(rows, row)
	}

	// 分页链接
	pv := func(page int) string {
		return base + "?" + listingQuery(a.sort, a.order, a.search, page)
	}
	hasPrev := a.page > 1
	hasNext := a.page < a.pageCount
	prevURL, nextURL := "", ""
	if hasPrev {
		prevURL = pv(a.page - 1)
	}
	if hasNext {
		nextURL = pv(a.page + 1)
	}

	// 搜索表单提交目标（保持 sort/order，重置到第一页，由表单隐藏域携带）
	totalLabel := "共 " + itoa(a.total) + " 项"
	if a.search != "" {
		totalLabel += "（匹配 “" + a.search + "”）"
	}

	displayPath := "/"
	if rel != "" {
		displayPath = "/" + rel
	}

	return pageData{
		Title:        s.cfg.Title,
		CurrentPath:  displayPath,
		Rel:          rel,
		Crumbs:       crumbs,
		ShowParent:   showParent,
		ParentHref:   parentHref,
		Files:        rows,
		Empty:        len(rows) == 0,
		Q:            a.search,
		Sort:         a.sort,
		Order:        a.order,
		SortURLs:     sortURLs,
		SearchAction: base,
		Page: pageInfo{
			Page:      a.page,
			PageCount: a.pageCount,
			Total:     a.total,
			PerPage:   s.pageSize,
			HasPrev:   hasPrev,
			HasNext:   hasNext,
			PrevURL:   prevURL,
			NextURL:   nextURL,
		},
		ZipHref:    base + "?zip=1",
		HasQuery:   a.search != "" || a.page > 1,
		TotalLabel: totalLabel,
	}
}

// nextOrder 决定点击某列表头后应使用的方向：
// 与当前排序列相同则翻转方向，否则（切换到新列）默认升序。
func nextOrder(curSort, targetSort, curOrder string) string {
	if curSort == targetSort {
		if curOrder == "desc" {
			return "asc"
		}
		return "desc"
	}
	return "asc"
}

// itoa 简易整数转字符串。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
