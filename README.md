<div align="center">

# GoLister

**A lightweight static file server with a clean, modern web UI.**

Browse, preview and download files from any directory — with dark mode, syntax highlighting,
ZIP download of whole folders, pagination and server-side search.
**Zero third-party Go dependencies** (standard library only).

[![Go Version](https://img.shields.io/badge/Go-1.24.3%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Dependencies](https://img.shields.io/badge/dependencies-0-brightgreen.svg)](go.mod)
[![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)](#)
[![GitHub stars](https://img.shields.io/github/stars/jiejiebiezheyang/golister?style=flat&logo=github)](https://github.com/jiejiebiezheyang/golister/stargazers)
[![Last commit](https://img.shields.io/github/last-commit/jiejiebiezheyang/golister)](https://github.com/jiejiebiezheyang/golister/commits/main)

**English** · [简体中文](README.zh-CN.md)

</div>

---

## Features

- 📁 **File browsing** — tree-style navigation, clickable breadcrumbs, and a dedicated `..` parent entry
- 🖼️ **Image lightbox** — zoom with the scroll wheel or buttons, previous/next, fit-to-window
- 📄 **Text preview with syntax highlighting** — ~36 common languages, oversized files truncated with a notice (>2 MiB)
- 📥 **File download** — safe attachment headers with RFC 5987 filename encoding
- 📦 **ZIP a whole folder** — streaming archive of any directory (hidden-file rules, symlinks skipped, 4 GiB cap)
- 🔍 **Server-side search** — filter by file name on Enter or with the search button
- 🔃 **Server-side sorting** — by name / size / modified time, ascending or descending, directories always first
- 🗂 **Pagination for large folders** — page size is configurable (200 by default)
- 🌙 **Dark mode** — follows the system preference, manual toggle is remembered (highlight theme switches too)
- 🚀 **Automatic port** — retries with the next port when the requested one is busy
- 🔒 **Path safety** — segment-by-segment decoding, rejects traversal, encoding tricks and symlink escapes
- 📱 **Responsive design** — adapts to any screen size

## Supported file types

### Images (lightbox preview)

`jpg` `jpeg` `png` `gif` `bmp` `webp` `svg` `ico` `avif`

### Text / code (text preview)

`txt` `md` `markdown` `json` `xml` `html` `css` `js` `mjs` `ts` `tsx` `jsx` `go` `py` `java` `c`
`cpp` `cc` `cxx` `h` `hpp` `sh` `bat` `cmd` `ps1` `yaml` `yml` `toml` `ini` `cfg` `conf` `log`
`sql` `rb` `php` `rs` `swift` `kt` `scala` `lua` `r` `dockerfile` `gitignore` `env` `csv` and more

### PDF / archives

`pdf` (download only); `zip` `rar` `7z` `tar` `gz` `tgz` `bz2` `xz` `zst` (download only; folder archiving is handled by the built-in ZIP packer)

## Build

```bash
# Requires Go 1.24.3+ (see go.mod)
cd golister
go build -o golister .

# Cross-compilation examples
GOOS=linux   GOARCH=amd64 go build -o golister-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o golister.exe .
```

## Usage

```bash
# Defaults: port 8080, share the current directory
./golister

# Custom port / directory / title / page size
./golister -port 18080 -dir /path/to/files -title "My Files"
./golister -port 8080 -dir ./public -page-size 500
```

Then open `http://localhost:8080`.

### Command-line flags

| Flag         | Description                            | Default               |
| ------------ | -------------------------------------- | --------------------- |
| `-port`      | Server port (auto-increments if taken) | `8080`                |
| `-dir`       | Directory to share                     | current directory `.` |
| `-title`     | Web page title                         | `cyan静态资源站`      |
| `-page-size` | Entries per page                       | `200`                 |
| `-h`         | Show help                              | –                     |

## How it works

### Browsing & navigation

- Whole table rows are clickable: folder rows open the folder, file rows open a preview when possible;
- The `..` row at the top and the breadcrumbs navigate upward; column headers toggle sorting (folders always first).

### Preview & download

- Text/code files open in an overlay without leaving the page, with automatic syntax highlighting;
- Images open in a lightbox — zoom with the scroll wheel, switch with ←/→ or the buttons;
- Previewable files preview on row click, while non-previewable files (or the row's **Download** button) download directly;
- Previews of very large files (>2 MiB) are truncated with a notice.

### Folder ZIP

- The **Pack ZIP** button in the top-right archives the current folder; every folder row also has a ZIP shortcut.
- The archive contains the folder name as its top-level entry; hidden files (leading `.`) and the server's own
  executable are excluded, and symlinks are always skipped (so nothing outside the shared root can leak into the archive).

### Search & pagination

- The search box performs a **server-side**, case-insensitive substring match on file names (the folder is fully matched before pagination);
- 200 entries per page by default (tunable with `-page-size`); sorting and search state are preserved across pages.

### Hidden entries

- Files and folders whose name starts with `.` are hidden from listings and archives;
- The server executable hides itself automatically;
- (Symlinks pointing outside the shared root are still rejected even when accessed by URL.)

### Dark mode

- The first visit follows the system `prefers-color-scheme`;
- Toggle with the 🌙/☀ button in the top-right; the choice is stored in `localStorage`.

## Project structure

| File          | Responsibility                                                                                    |
| ------------- | ------------------------------------------------------------------------------------------------- |
| `main.go`     | Flag parsing, startup, graceful shutdown (SIGINT/SIGTERM)                                         |
| `server.go`   | `Server`/`NewServer`, routing, directory pages and ZIP dispatch                                   |
| `listing.go`  | Directory listing, hidden-entry filtering, sorting                                                |
| `pathsec.go`  | URL decoding and traversal checks, symlink containment validation                                 |
| `filekind.go` | Extension classification and MIME types                                                           |
| `serve.go`    | File download / inline images / text preview (with truncation)                                    |
| `zipdir.go`   | Streaming ZIP archiving of directories                                                            |
| `page.go`     | Template data assembly (breadcrumbs, pagination, sort links)                                      |
| `helpers.go`  | Size/time formatting and query-string helpers                                                     |
| `index.html`  | The single page template: **all HTML/SVG/CSS/JS lives here** (Go source contains no HTML strings) |

> Note: none of the `*.go` sources contain HTML strings. Every page markup, icon SVG, style and
> script lives in the `index.html` template and is embedded into the binary via `go:embed`.
> `highlight.min.js` is loaded at runtime from BootCDN and is **not** bundled
> (for fully offline deployments, drop the file into the project and inline/reference it in the template).

## Development

```bash
go test ./...   # unit tests + httptest integration tests
go vet ./...
gofmt -w .
```

## Tech stack

- Go standard library (`net/http`, `embed`, `html/template`, `archive/zip`, `mime`, …) — **zero third-party Go dependencies**
- Vanilla HTML/CSS/JavaScript (no frameworks)
- Syntax highlighting: [highlight.js](https://highlightjs.org/) v11.11.1 via BootCDN
  (highlighting needs network access; when the CDN is unavailable, the preview gracefully falls back to plain text)

## License

[MIT](LICENSE) © 2026 jiejiebiezheyang
