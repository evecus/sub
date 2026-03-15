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
	"github.com/evecus/sub/internal/auth"
	"github.com/evecus/sub/internal/store"
)

//go:embed web/dist
var webFS embed.FS

func main() {
	// ── Flags ─────────────────────────────────────────────────────────────────
	port    := flag.String("port", "8080", "监听端口")
	dataDir := flag.String("dir",  "",     "数据目录（默认 ~/.sub-store）")
	authArg := flag.String("auth", "",     "登录凭证，格式：用户名:密码")
	flag.Parse()

	// ── Credentials ───────────────────────────────────────────────────────────
	username := "admin"
	password := ""
	firstRun := false

	if *authArg != "" {
		parts := strings.SplitN(*authArg, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			log.Fatal("--auth 格式错误，应为 用户名:密码，例如：--auth admin:mypassword")
		}
		username = parts[0]
		password = parts[1]
	} else {
		password = auth.GeneratePassword()
		firstRun = true
	}

	authMgr := auth.New(username, password)

	// ── Store ─────────────────────────────────────────────────────────────────
	s, err := store.New(*dataDir)
	if err != nil {
		log.Fatalf("初始化数据目录失败: %v", err)
	}

	// ── Router ────────────────────────────────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	authMgr.RegisterAuthRoutes(r)

	h := api.New(s)
	r.GET("/sub/:token", h.ServeSubscription)

	protected := r.Group("/api", authMgr.Middleware())
	h.RegisterProtectedRoutes(protected)

	// ── Frontend ──────────────────────────────────────────────────────────────
	distFS, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		log.Fatalf("加载前端资源失败: %v", err)
	}
	fileServer := http.FileServer(http.FS(distFS))
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if _, err := fs.Stat(distFS, p[1:]); err != nil {
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	// ── Banner ────────────────────────────────────────────────────────────────
	dataPath := s.DataDir()
	fmt.Println()
	fmt.Printf("┌──────────────────────────────────────────────────────┐\n")
	fmt.Printf("│  Sub-Store %-42s│\n", buildinfo.Version)
	fmt.Printf( "│  🚀 Sub-Store  →  http://localhost:%-18s│\n", *port+"  ")
	fmt.Println("├──────────────────────────────────────────────────────┤")
	fmt.Printf( "│  👤 用户名  :  %-36s│\n", username)
	if firstRun {
		fmt.Printf("│  🔑 密  码  :  %-36s│\n", password)
		fmt.Println("│                                                      │")
		fmt.Println("│  ⚠️  随机密码，建议使用 --auth 参数固定              │")
	} else {
		fmt.Println("│  🔑 密  码  :  (已通过 --auth 参数设置)              │")
	}
	fmt.Println("├──────────────────────────────────────────────────────┤")
	fmt.Printf( "│  📁 数据目录:  %-36s│\n", dataPath)
	fmt.Println("└──────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("用法示例:")
	fmt.Println("  ./sub-store                              # 默认端口 8080，随机密码")
	fmt.Println("  ./sub-store --port 9090                  # 自定义端口")
	fmt.Println("  ./sub-store --auth admin:mypassword      # 固定密码")
	fmt.Println("  ./sub-store --dir /data/sub-store        # 自定义数据目录")
	fmt.Println("  ./sub-store --port 9090 --auth admin:123 --dir /data/sub-store")
	fmt.Println()

	if err := r.Run(":" + *port); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
