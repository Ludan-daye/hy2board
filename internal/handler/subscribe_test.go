package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestShadowrocketSubscriptionFormatReturnsNativeNodeLines(t *testing.T) {
	router, user := setupSubscribeTest(t)

	res := performSubscribeRequest(router, "/api/sub/"+user.SubToken+"?format=shadowrocket", "")
	if res.Code != http.StatusOK {
		t.Fatalf("shadowrocket status = %d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.HasPrefix(body, "HK1-plain=hysteria2,203.0.113.10,443,auth=alice-secret,udp=1,peer=bing.com,alpn=h3,skip-cert-verify=true,insecure=1") {
		t.Fatalf("shadowrocket subscription should start with a native node line, got:\n%s", body)
	}
	if !strings.Contains(body, "HK1-obfs=hysteria2,203.0.113.20,8443,auth=alice-secret,obfsParam=obfs-pass,udp=1,peer=bing.com,alpn=h3,skip-cert-verify=true,insecure=1") {
		t.Fatalf("shadowrocket subscription should contain native obfs node line, got:\n%s", body)
	}
	for _, unexpected := range []string{"[General]", "[Proxy]", "hysteria2://", " = hysteria2", ", ", "obfs-password", "obfs=salamander"} {
		if strings.Contains(body, unexpected) {
			t.Fatalf("shadowrocket node subscription should not contain %q:\n%s", unexpected, body)
		}
	}

	conf := performSubscribeRequest(router, "/api/sub/"+user.SubToken+"?format=shadowrocket-conf", "")
	if conf.Code != http.StatusOK {
		t.Fatalf("shadowrocket-conf status = %d body=%s", conf.Code, conf.Body.String())
	}
	confBody := conf.Body.String()
	for _, want := range []string{
		"[General]",
		"[Proxy]",
		"HK1-plain=hysteria2,203.0.113.10,443,auth=alice-secret,udp=1,peer=bing.com,alpn=h3,skip-cert-verify=true,insecure=1",
		"HK1-obfs=hysteria2,203.0.113.20,8443,auth=alice-secret,obfsParam=obfs-pass,udp=1,peer=bing.com,alpn=h3,skip-cert-verify=true,insecure=1",
		"[Rule]",
		"DOMAIN-SUFFIX,polymarket.com,HongKong",
	} {
		if !strings.Contains(confBody, want) {
			t.Fatalf("shadowrocket-conf should contain %q:\n%s", want, confBody)
		}
	}
	for _, unexpected := range []string{" = hysteria2", "obfs-password", "obfs=salamander"} {
		if strings.Contains(confBody, unexpected) {
			t.Fatalf("shadowrocket-conf should not contain %q:\n%s", unexpected, confBody)
		}
	}
}

func TestShadowrocketUserAgentReturnsNativeNodeLines(t *testing.T) {
	router, user := setupSubscribeTest(t)

	res := performSubscribeRequest(router, "/api/sub/"+user.SubToken, "Shadowrocket/2.2.59 CFNetwork")
	if res.Code != http.StatusOK {
		t.Fatalf("shadowrocket user-agent status = %d body=%s", res.Code, res.Body.String())
	}
	if !strings.HasPrefix(res.Body.String(), "HK1-plain=hysteria2,203.0.113.10,443,auth=alice-secret,udp=1,peer=bing.com,alpn=h3,skip-cert-verify=true,insecure=1") {
		t.Fatalf("shadowrocket user-agent should receive native node lines, got:\n%s", res.Body.String())
	}
}

func setupSubscribeTest(t *testing.T) (*gin.Engine, model.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	oldDB := database.DB
	oldConfig := config.C
	t.Cleanup(func() {
		database.DB = oldDB
		config.C = oldConfig
	})
	config.C = config.Config{}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Node{}, &model.CustomRoutingRule{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db

	user := model.User{
		Username:      "alice",
		Hy2Password:   "alice-secret",
		SubToken:      "sub-token",
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		Enabled:       true,
		NodeIDs:       "all",
		RuleAI:        false,
		RuleStreaming: false,
		RuleChina:     true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	plainNode := model.Node{
		Name:          "HK1-plain",
		Host:          "203.0.113.10",
		Port:          443,
		SNI:           "bing.com",
		Insecure:      true,
		Healthy:       true,
		SortOrder:     1,
		TrafficAPI:    "http://traffic.local",
		TrafficSecret: "secret",
	}
	if err := db.Create(&plainNode).Error; err != nil {
		t.Fatalf("create plain node: %v", err)
	}

	obfsNode := model.Node{
		Name:          "HK1-obfs",
		Host:          "203.0.113.20",
		Port:          8443,
		SNI:           "bing.com",
		Insecure:      true,
		ObfsType:      "salamander",
		ObfsPassword:  "obfs-pass",
		Healthy:       true,
		SortOrder:     2,
		TrafficAPI:    "http://traffic-obfs.local",
		TrafficSecret: "obfs-secret",
	}
	if err := db.Create(&obfsNode).Error; err != nil {
		t.Fatalf("create obfs node: %v", err)
	}

	rule := model.CustomRoutingRule{
		Enabled:   true,
		Name:      "Polymarket HK",
		Kind:      "DOMAIN-SUFFIX",
		Value:     "polymarket.com",
		Policy:    "HongKong",
		SortOrder: 30,
	}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatalf("create routing rule: %v", err)
	}

	r := gin.New()
	r.GET("/api/sub/:token", Subscribe)
	return r, user
}

func performSubscribeRequest(r http.Handler, path string, userAgent string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}
