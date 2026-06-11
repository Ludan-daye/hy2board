package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/handler"
	"github.com/ludandaye/hy2board/internal/middleware"
	"github.com/ludandaye/hy2board/internal/service"
)

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	if err := config.Load(cfgPath); err != nil {
		log.Fatalf("Config: %v", err)
	}

	if err := database.Init(config.C.Database.Path); err != nil {
		log.Fatalf("Database: %v", err)
	}

	service.StartHealthChecker(30 * time.Second)
	service.StartTrafficCache(3 * time.Second)
	service.StartAutoResetScheduler(60 * time.Second)
	service.StartUserBucketCache(30 * time.Second)
	service.StartTrafficLogger(60 * time.Second)
	service.StartTelegramBot()
	service.StartDailyPoster()
	service.StartWeeklyLeaderboard()
	service.StartStaticIPProber(2 * time.Minute)

	r := gin.Default()
	r.Use(gzip.Gzip(
		gzip.DefaultCompression,
		gzip.WithExcludedExtensions([]string{".png", ".jpg", ".gif", ".ico", ".woff2"}),
		gzip.WithExcludedPaths([]string{"/status"}),
	))

	// Cache static assets (JS/CSS have content hash in filename, safe to cache long)
	r.Use(func(c *gin.Context) {
		if p := c.Request.URL.Path; len(p) >= 8 && p[:8] == "/assets/" {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	})

	// Status page reverse proxy → localhost:3000
	statusTarget, _ := url.Parse("http://127.0.0.1:3000")
	statusProxy := httputil.NewSingleHostReverseProxy(statusTarget)
	statusProxy.Director = func(req *http.Request) {
		req.URL.Scheme = statusTarget.Scheme
		req.URL.Host = statusTarget.Host
		req.Host = statusTarget.Host
		req.Header.Del("Accept-Encoding")
	}
	proxyToStatus := func(c *gin.Context) {
		statusProxy.ServeHTTP(c.Writer, c.Request)
	}
	r.Any("/status", proxyToStatus)
	r.Any("/status/*path", proxyToStatus)

	// Public endpoints
	r.GET("/api/sub/:token", handler.Subscribe)
	r.HEAD("/api/sub/:token", handler.Subscribe)
	r.POST("/api/auth/hy2", handler.Hy2Auth)
	r.GET("/api/downloads", handler.ListDownloads)
	r.Static("/dl", "./downloads")
	r.GET("/api/app/version", handler.AppVersion)
	r.GET("/api/app/features", handler.AppFeatures)
	r.GET("/api/app/downloads", handler.ListDownloads)

	// Auth
	r.POST("/api/admin/login", handler.Login)
	r.POST("/api/user/login", handler.UserLogin)

	// Admin API (JWT protected)
	admin := r.Group("/api/admin", middleware.AdminAuth())
	{
		admin.GET("/stats", handler.GetStats)
		admin.GET("/history", handler.GetHistory)
		admin.GET("/buckets", handler.GetBuckets)
		admin.GET("/audit-logs", handler.ListAuditLogs)

		admin.GET("/routing-rules", handler.ListRoutingRules)
		admin.POST("/routing-rules", handler.CreateRoutingRule)
		admin.PUT("/routing-rules/:id", handler.UpdateRoutingRule)
		admin.DELETE("/routing-rules/:id", handler.DeleteRoutingRule)
		admin.POST("/routing-rules/preview", handler.PreviewRoutingRule)

		admin.GET("/nodes", handler.ListNodes)
		admin.POST("/nodes", handler.CreateNode)
		admin.PUT("/nodes/:id", handler.UpdateNode)
		admin.DELETE("/nodes/:id", handler.DeleteNode)

		admin.GET("/users", handler.ListUsers)
		admin.GET("/users/:id", handler.GetUser)
		admin.POST("/users", handler.CreateUser)
		admin.POST("/users/bulk", handler.BulkUsers)
		admin.PUT("/users/:id", handler.UpdateUser)
		admin.DELETE("/users/:id", handler.DeleteUser)
		admin.POST("/users/:id/reset-token", handler.ResetSubToken)
		admin.POST("/users/:id/renew", handler.RenewUser)
		admin.POST("/users/:id/set-password", handler.UserSetPassword)
		admin.POST("/users/:id/toggle", handler.ToggleUser)
		admin.POST("/users/:id/toggle-chain", handler.ToggleChainProxy)
		admin.POST("/users/:id/reset-traffic", handler.ResetTraffic)

		admin.GET("/plans", handler.ListPlans)
		admin.GET("/plans/:id", handler.GetPlan)
		admin.POST("/plans", handler.CreatePlan)
		admin.PUT("/plans/:id", handler.UpdatePlan)
		admin.DELETE("/plans/:id", handler.DeletePlan)
		admin.POST("/plans/:id/apply-to/:userId", handler.ApplyPlanToUser)
		admin.POST("/plans/:id/set-proxy-password", handler.SetProxyPassword)

		admin.GET("/static-ips", handler.ListStaticIPs)
		admin.GET("/payments", handler.ListPayments)
		admin.GET("/payments/summary", handler.SummaryPayments)
		admin.PUT("/payments/:id", handler.UpdatePayment)
		admin.DELETE("/payments/:id", handler.DeletePayment)
		admin.GET("/payments.csv", handler.ExportPaymentsCSV)

		admin.GET("/costs", handler.ListCosts)
		admin.POST("/costs", handler.CreateCost)
		admin.PUT("/costs/:id", handler.UpdateCost)
		admin.DELETE("/costs/:id", handler.DeleteCost)

		admin.GET("/users/:id/qrcode", handler.GetSubscriptionQRCode)
		admin.GET("/users/:id/subscription-urls", handler.GetSubscriptionURLs)
		admin.GET("/users/:id/traffic-history", handler.GetUserTrafficHistory)

		admin.GET("/tg/status", handler.GetTelegramStatus)
		admin.POST("/tg/test-admin-notice", handler.SendTelegramAdminTestNotice)
		admin.POST("/tg/test-post", handler.SendTelegramTestPost)
		admin.POST("/tg/announce-activity", handler.SendTelegramActivityAnnouncement)
	}

	// User API (JWT protected)
	user := r.Group("/api/user", middleware.UserAuth())
	{
		user.GET("/me", handler.UserMe)
		user.GET("/session", handler.ClientSession)
		user.GET("/app/bootstrap", handler.ClientBootstrap)
		user.GET("/profile", handler.ClientProfile)
		user.GET("/plan", handler.ClientPlan)
		user.GET("/traffic/summary", handler.ClientTrafficSummary)
		user.GET("/traffic/history", handler.ClientTrafficHistory)
		user.GET("/traffic/nodes", handler.ClientTrafficNodes)
		user.GET("/nodes", handler.ClientNodes)
		user.GET("/client-config", handler.ClientConfig)
		user.GET("/announcements", handler.ClientAnnouncements)
		user.GET("/help", handler.ClientHelp)
		user.GET("/diagnostics", handler.ClientDiagnostics)
	}

	// Serve frontend
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/favicon.ico", "./web/dist/favicon.ico")
	r.StaticFile("/favicon.svg", "./web/dist/favicon.svg")
	r.StaticFile("/icons.svg", "./web/dist/icons.svg")
	r.NoRoute(func(c *gin.Context) {
		// Cache SPA HTML briefly at edge; revalidation is free since assets are immutable.
		c.Header("Cache-Control", "public, max-age=60, s-maxage=300")
		c.File("./web/dist/index.html")
	})

	log.Printf("hy2board starting on %s", config.C.Server.Listen)
	r.Run(config.C.Server.Listen)
}
