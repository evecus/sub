package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/evecus/sub/internal/api"
	"github.com/evecus/sub/internal/buildinfo"
	"github.com/evecus/sub/internal/store"
)

//go:embed web/dist
var webFS embed.FS

func main() {
	// ── Flags ─────────────────────────────────────────────────────────────────
	port    := flag.String("port", "8080", "监听端口")
	dataDir := flag.String("dir",  "",     "数据目录（默认 ~/.sub-store）")
	path    := flag.String("path", "/",    "后端路径前缀，例如 --path=/secret123")
	flag.Parse()

	// 规范化 path：必须以 / 开头，不以 / 结尾
	backendPath := *path
	if !strings.HasPrefix(backendPath, "/") {
		backendPath = "/" + backendPath
	}
	backendPath = strings.TrimRight(backendPath, "/")
	if backendPath == "" {
		backendPath = "/"
	}

	// ── Store ─────────────────────────────────────────────────────────────────
	s, err := store.New(*dataDir)
	if err != nil {
		log.Fatalf("初始化数据目录失败: %v", err)
	}

	// ── Router ────────────────────────────────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Content-Type"},
	}))

	h := api.New(s, backendPath)

	// ── 所有路由挂在 backendPath 前缀下 ──────────────────────────────────────
	var group *gin.RouterGroup
	if backendPath == "/" {
		group = r.Group("/")
	} else {
		group = r.Group(backendPath)
	}

	// 所有路由（API + 下载 + 订阅）
	h.RegisterRoutes(group)

	// ── Frontend（静态文件，挂在根路径）────────────────────────────────────
	distFS, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		log.Fatalf("加载前端资源失败: %v", err)
	}
	fileServer := http.FileServer(http.FS(distFS))
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		// API 请求没有匹配路由 → 404（不能返回 index.html，否则前端会误判为成功）
		if strings.HasPrefix(p, "/api/") || strings.Contains(p, "/api/") {
			c.JSON(404, gin.H{"status": "failed", "error": gin.H{"message": "not found"}})
			return
		}
		// 静态资源直接返回
		if _, err := fs.Stat(distFS, strings.TrimPrefix(p, "/")); err == nil {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		// 其他所有路径返回 index.html（SPA）
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// ── Banner ────────────────────────────────────────────────────────────────
	dataPath := s.DataDir()
	fmt.Println()
	fmt.Println("┌──────────────────────────────────────────────────────┐")
	fmt.Printf( "│  Sub-Store %-42s│\n", buildinfo.Version)
	fmt.Printf( "│  🚀 地址  →  http://localhost:%s%-14s│\n", *port, "")
	fmt.Println("├──────────────────────────────────────────────────────┤")
	fmt.Printf( "│  🔑 后端路径:  %-36s│\n", backendPath)
	fmt.Printf( "│  📁 数据目录:  %-36s│\n", dataPath)
	fmt.Println("└──────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("用法示例:")
	fmt.Println("  ./sub-store                                  # 无路径保护")
	fmt.Println("  ./sub-store --path=/secret123                # 设置后端路径")
	fmt.Println("  ./sub-store --port=9090 --path=/secret123    # 自定义端口+路径")
	fmt.Println("  ./sub-store --path=/secret123 --dir=/data    # 自定义数据目录")
	fmt.Println()
	fmt.Printf("  访问 http://localhost:%s/ 输入后端路径 %s 即可进入\n", *port, backendPath)
	fmt.Println()

	if err := r.Run(":" + *port); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
