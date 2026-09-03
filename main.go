package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := parseFlags()
	run(cfg)
}

func run(cfg Config) {
	srv, err := NewServer(cfg)
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}

	// 端口占用自动递增重试（保留原行为）
	port := cfg.Port
	var ln net.Listener
	for {
		addr := fmt.Sprintf(":%d", port)
		l, err := net.Listen("tcp", addr)
		if err == nil {
			ln = l
			break
		}
		log.Printf("端口 %d 被占用，尝试 %d...", port, port+1)
		port++
	}

	httpSrv := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 优雅退出
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("服务器启动成功: http://localhost:%d （共享目录: %s）", port, srv.rootAbs)
		errCh <- httpSrv.Serve(ln)
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("服务异常退出: %v", err)
		}
	case <-ctx.Done():
		log.Println("收到退出信号，正在关闭...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			log.Printf("关闭超时: %v", err)
		}
	}
}

// parseFlags 解析命令行参数。
func parseFlags() Config {
	var cfg Config

	fs := flag.NewFlagSet("golister", flag.ExitOnError)
	fs.IntVar(&cfg.Port, "port", 8080, "服务器端口（默认8080），被占用时自动递增")
	fs.StringVar(&cfg.Dir, "dir", ".", "共享目录（默认当前目录）")
	fs.StringVar(&cfg.Title, "title", "cyan静态资源站", "网页标题")
	fs.IntVar(&cfg.PageSize, "page-size", defaultPageSize, "列表每页条目数（大目录性能优化）")
	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprintln(out, "GoLister - 轻量静态文件浏览/下载服务器")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "用法: golister [选项]")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "选项:")
		fs.PrintDefaults()
		fmt.Fprintln(out)
		fmt.Fprintln(out, "示例:")
		fmt.Fprintln(out, "  golister -port 18080 -dir /path/to/files -title 我的资源站")
		fmt.Fprintln(out, "  golister -port 8080 -dir . -page-size 500")
	}
	_ = fs.Parse(os.Args[1:])

	if cfg.Dir == "" {
		cfg.Dir = "."
	}
	if cfg.Title == "" {
		cfg.Title = "cyan静态资源站"
	}
	cfg.Log = log.New(os.Stderr, "", log.LstdFlags)
	return cfg
}
