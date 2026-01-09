# GoLister

一个简单易用的静态文件服务器，提供美观的 Web 界面用于浏览、预览和下载文件。

## 功能特性

- 📁 **文件浏览** - 以树形结构展示目录和文件
- 🖼️ **图片预览** - 支持常见图片格式的在线预览
- 📄 **文本预览** - 支持代码和文本文件的在线查看
- 📥 **文件下载** - 一键下载任意文件
- 🔍 **智能排序** - 支持按名称、大小、修改时间排序
- 🎨 **美观界面** - 清晰的文件图标和颜色区分
- 🔧 **灵活配置** - 支持自定义端口、目录和标题
- 🚀 **自动端口** - 端口被占用时自动递增重试
- 📱 **响应式设计** - 适配各种屏幕尺寸

## 支持的文件类型

### 图片格式

- jpg, jpeg, png, gif, bmp, webp, svg

### 文本格式

- txt, md, json, xml, html, css, js, log

### 代码格式

- go, py, java, c, cpp, h, hpp, sh, bat, yaml, yml, toml, ini, conf, ts, tsx, jsx, rs, rb, php, swift, kt, scala

## 安装

### 从源码编译

```bash
# 克隆或下载项目
cd golister

# 编译
go build -o golister main.go
```

### 交叉编译

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o golister-linux-amd64 main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o golister.exe main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o golister-darwin-amd64 main.go
```

## 使用方法

### 基本使用

```bash
# 使用默认配置（端口8080，当前目录）
./golister

# 指定端口
./golister -port 18080

# 指定共享目录
./golister -dir /path/to/files

# 指定网页标题
./golister -title "我的资源站"

# 组合使用
./golister -port 18080 -dir /path/to/files -title "我的资源站"
```

### 命令行参数

| 参数     | 说明         | 默认值          |
| -------- | ------------ | --------------- |
| `-port`  | 服务器端口   | 8080            |
| `-dir`   | 共享目录     | 当前目录        |
| `-title` | 网页标题     | cyan 静态资源站 |
| `-help`  | 显示帮助信息 | -               |

### 查看帮助

```bash
./golister -help
```

## 功能说明

### 文件预览

- 点击图片或文本文件可在线预览
- 预览模式会直接在浏览器中显示文件内容

### 文件下载

- 点击任意文件即可下载
- 下载时会自动设置正确的文件名

### 文件图标

- 📁 蓝色文件夹图标
- 🖼️ 绿色图片图标
- 📄 橙色文本/代码图标
- 📕 红色 PDF 图标
- 📦 紫色压缩包图标
- 📄 灰色默认文件图标

### 自动隐藏

- 自动隐藏以 `.` 开头的隐藏文件
- 自动隐藏可执行文件本身

## 示例

### 启动本地文件服务器

```bash
./golister -port 8080 -dir ~/Documents -title "文档中心"
```

访问 `http://localhost:8080` 即可浏览文件。

### 共享项目目录

```bash
./golister -port 9000 -dir ./public -title "项目资源"
```

访问 `http://localhost:9000` 查看共享文件。

## 技术栈

- Go 1.16+
- 标准库（net/http, embed, template 等）
- 无第三方依赖

## 许可证

MIT License
