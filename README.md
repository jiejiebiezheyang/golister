# GoLister

一个简单易用的静态文件服务器：提供美观的 Web 界面用于**浏览、预览、下载**文件，
并支持**暗色主题、语法高亮、目录 ZIP 打包下载、大目录分页与服务端搜索**。Go 侧**零第三方依赖**（纯标准库）；
前端语法高亮通过 BootCDN 引入 [highlight.js](https://highlightjs.org/)（可随时替换为本地文件）。

## 功能特性

- 📁 **文件浏览** — 树形目录导航 + 可点击面包屑 + “上级目录”入口
- 🖼️ **图片大图浏览** — 灯箱预览：缩放（滚轮/按钮）、上一张/下一张、适应窗口
- 📄 **文本预览 + 语法高亮** — 代码文件自动高亮（约 36 种常见语言），超大文件截断（>2 MiB 提示）
- 📥 **文件下载** — 安全的附件头（文件名自动 RFC 5987 编码）
- 📦 **目录 ZIP 下载** — 任意目录流式打包（含隐藏规则，跳过符号链接，4 GiB 上限）
- 🔍 **服务端搜索** — 回车或点击搜索按钮后按文件名过滤
- 🔃 **服务端排序** — 按 名称/大小/修改时间 升/降序，目录始终置顶
- 🗂 **大目录分页** — 每页条目数可配置（默认 200）
- 🌙 **暗色主题** — 跟随系统偏好，可手动切换并记忆（高亮主题同步切换）
- 🚀 **自动端口** — 端口被占用时自动递增重试
- 🔒 **路径安全** — 逐段解码校验、拒绝穿越/编码混淆/符号链接逃逸
- 📱 **响应式设计** — 适配各种屏幕尺寸

## 支持的文件类型

### 图片（灯箱预览）

jpg, jpeg, png, gif, bmp, webp, svg, ico, avif

### 文本 / 代码（文本预览）

txt, md, markdown, json, xml, html, css, js, mjs, ts, tsx, jsx, go, py, java, c,
cpp, cc, cxx, h, hpp, sh, bat, cmd, ps1, yaml, yml, toml, ini, cfg, conf, log,
sql, rb, php, rs, swift, kt, scala, lua, r, dockerfile, gitignore, env, csv 等

### PDF / 压缩包

pdf（仅下载）；zip, rar, 7z, tar, gz, bz2, xz 等（仅下载，目录用 ZIP 打包）

## 安装 / 构建

```bash
# 需要 Go 1.21+（go.mod 声明 go 1.24.3）
cd golister
go build -o golister .
# 交叉编译示例
GOOS=linux GOARCH=amd64 go build -o golister-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o golister.exe .
```

## 使用方法

```bash
# 默认配置：端口 8080，共享当前目录
./golister

# 指定端口 / 目录 / 标题 / 每页条数
./golister -port 18080 -dir /path/to/files -title "我的资源站"
./golister -port 8080 -dir ./public -page-size 500
```

启动后访问 `http://localhost:8080`。

### 命令行参数

| 参数         | 说明                    | 默认值          |
| ------------ | ----------------------- | --------------- |
| `-port`      | 服务器端口（被占用自动递增） | 8080            |
| `-dir`       | 共享目录                | 当前目录 `.`    |
| `-title`     | 网页标题                | cyan 静态资源站 |
| `-page-size` | 列表每页条目数          | 200             |
| `-h`         | 查看帮助                | -               |

## 功能说明

### 浏览与导航

- 表格中**整行可点击**：目录行点击进入该目录；文件行有预览能力时优先打开预览；
- 表格顶部“`..`”行与面包屑可回上级；表头可点击排序（目录永远在最前）。

### 预览与下载

- 文本/代码文件在弹层中查看（不离开页面），代码自动语法高亮；
- 图片打开灯箱，可滚轮缩放、←/→ 或按钮切换当前页图片；
- 可预览文件的整行/名称默认预览，行内“下载”按钮或不可预览文件整行点击即下载；
- 超大文件（>2 MiB）的预览会被截断并提示。

### 目录 ZIP

- 列表页右上角“打包 ZIP”打包当前目录；每个目录行也有 ZIP 快捷按钮。
- 归档顶层包含目录名本身；隐藏文件（`.` 开头）与服务器自身可执行文件不入包，
  符号链接一律跳过（防止把共享目录外的目标文件打包出去）。

### 搜索与分页

- 搜索框按文件名做**服务端**不区分大小写的子串过滤（整目录匹配后再分页）；
- 每页默认 200 项（`-page-size` 可调），支持翻页，排序/搜索状态在翻页间保持。

### 隐藏规则

- 以 `.` 开头的文件/目录在列表与 ZIP 中隐藏；
- 服务器可执行文件自身自动隐藏。
- （直接输入 URL 访问隐藏文件仍会被拒绝的场景：符号链接指向共享目录之外的路径。）

### 暗色主题

- 首次访问跟随系统 `prefers-color-scheme`；
- 右上角 🌙/☀ 切换，选择保存在浏览器 localStorage 中。

## 代码结构

| 文件           | 职责                                              |
| -------------- | ------------------------------------------------- |
| `main.go`      | 参数解析、启动、优雅退出（SIGINT/SIGTERM）        |
| `config.go`*   | （内置于 main.go）配置定义                         |
| `server.go`    | Server/NewServer、路由、目录页/打包分发           |
| `listing.go`   | 目录列举、隐藏过滤、排序                          |
| `pathsec.go`   | URL 解码与防穿越校验、符号链接包含校验            |
| `filekind.go`  | 扩展名统一归类、MIME 类型                         |
| `serve.go`     | 文件下载 / 图片内联 / 文本预览（截断）            |
| `zipdir.go`    | 目录流式 ZIP 打包                                 |
| `page.go`      | 列表页模板数据组装（面包屑、分页、排序链接）      |
| `helpers.go`   | 大小/时间格式化、查询串构造                       |
| `index.html`   | 唯一页面模板：**全部 HTML/SVG/CSS/JS 均在此文件**（Go 源码零 HTML 字符串） |

> 说明：`*.go` 源文件中**不包含任何 HTML 字符串**。所有页面标记、图标 SVG、
> 样式与脚本都位于 `index.html` 模板内，由 `go:embed` 打包进二进制。
> `highlight.min.js` 由页面通过 BootCDN 动态加载，**不会**打进二进制
> （如需完全离线部署，可将文件放入项目并改为在模板中内联/静态引用）。

## 开发

```bash
go test ./...   # 单元测试 + httptest 集成测试
go vet ./...
gofmt -w .
```

## 技术栈

- Go 标准库（net/http, embed, html/template, archive/zip, mime 等）——**零第三方 Go 依赖**
- 原生 HTML/CSS/JavaScript（无框架）
- 前端语法高亮：[highlight.js](https://highlightjs.org/) v11.11.1，经 BootCDN 加载
  （高亮需联网；CDN 不可用时文本预览自动回退为纯文本，不影响其他功能）

## 许可证

MIT License
