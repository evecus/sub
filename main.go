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
	"github.com/evecus/sub/internal/ratelimit"
	"github.com/evecus/sub/internal/scheduler"
	"github.com/evecus/sub/internal/session"
	"github.com/evecus/sub/internal/store"
)

//go:embed web/dist
var webFS embed.FS

func main() {
	// ── Flags ─────────────────────────────────────────────────────────────────
	port    := flag.String("port", "8080", "监听端口")
	dataDir := flag.String("dir",  "",     "数据目录（默认 ~/.sub-store）")
	path    := flag.String("path", "",     "后端路径前缀，例如 --path=/secret123（必填）")
	flag.Parse()

	// path 必填
	if *path == "" {
		fmt.Println("错误：必须指定 --path 参数")
		fmt.Println("示例：./sub-store --path=/secret123 --port=8080 --dir=/data")
		log.Fatal("缺少 --path 参数，拒绝启动")
	}

	// 规范化 path
	backendPath := *path
	if !strings.HasPrefix(backendPath, "/") {
		backendPath = "/" + backendPath
	}
	backendPath = strings.TrimRight(backendPath, "/")

	// ── Store ─────────────────────────────────────────────────────────────────
	s, err := store.New(*dataDir)
	if err != nil {
		log.Fatalf("初始化数据目录失败: %v", err)
	}

	// ── Rate Limiter ──────────────────────────────────────────────────────────
	rl := ratelimit.New(s.DataDir())

	// ── Router ────────────────────────────────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)

	// 只打印错误日志，不打印每条请求
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Cookie"},
		AllowCredentials: true,
	}))

	h := api.New(s, backendPath)

	// ── API 路由（挂在 backendPath/api 下）────────────────────────────────────
	apiGroup := r.Group(backendPath + "/api")
	// utils/env 是路径验证入口，加防爆破保护；成功后由 GetEnv 内部设置 session cookie
	apiGroup.GET("/utils/env", rl.PathGuard(), h.GetEnv)
	// 其他 API 路由需要有效的当天 session
	authAPI := r.Group(backendPath+"/api", session.Guard())
	h.RegisterRoutes(authAPI)

	// ── 公开路由（不含 backendPath，不暴露管理路径）────────────────────────
	// /sub/:token 加防爆破保护
	r.GET("/sub/:token", rl.SubGuard(), h.ServeSubscription)

	// download 路由挂在 backendPath 下（已知路径才能下载）
	pubGroup := r.Group(backendPath)
	pubGroup.GET("/download/:name", h.DownloadSub)
	pubGroup.GET("/download/collection/:name", h.DownloadCollectionSub)

	// ── Frontend ──────────────────────────────────────────────────────────────
	distFS, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		log.Fatalf("加载前端资源失败: %v", err)
	}
	fileServer := http.FileServer(http.FS(distFS))
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		// API 路径没匹配到路由 → 404 JSON
		if strings.Contains(p, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"status": "failed", "error": gin.H{"message": "not found"}})
			return
		}
		// 静态资源直接返回
		if _, err := fs.Stat(distFS, strings.TrimPrefix(p, "/")); err == nil {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		// SPA fallback
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// ── Banner ────────────────────────────────────────────────────────────────
	dataPath := s.DataDir()
	fmt.Println()
	fmt.Println("┌──────────────────────────────────────────────────────┐")
	fmt.Printf( "│  Sub-Store %-42s│\n", buildinfo.Version)
	fmt.Printf( "│  🚀 地址     →  http://localhost:%s%-13s│\n", *port, "")
	fmt.Println("├──────────────────────────────────────────────────────┤")
	fmt.Printf( "│  🔑 后端路径 :  %-36s│\n", backendPath)
	fmt.Printf( "│  📁 数据目录 :  %-36s│\n", dataPath)
	fmt.Println("└──────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Printf("  访问 http://localhost:%s/ 输入后端路径 %s 即可进入\n\n", *port, backendPath)

	if err := r.Run(":" + *port); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
