package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestVlessClientsReturnsOnlyActiveUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.User{})
	database.DB = db
	config.C.Node.Secret = "s3cret"

	future := time.Now().Add(24 * time.Hour)
	db.Create(&model.User{Username: "alice", Hy2Password: "a:x", SubToken: "t1", Enabled: true, ExpiresAt: future})
	// disabled — set via explicit Update like ToggleUser does (GORM default:true
	// would otherwise override an explicit false at Create).
	bob := model.User{Username: "bob", Hy2Password: "b:x", SubToken: "t2", Enabled: true, ExpiresAt: future}
	db.Create(&bob)
	db.Model(&bob).Update("enabled", false)
	db.Create(&model.User{Username: "carol", Hy2Password: "c:x", SubToken: "t3", Enabled: true, TrafficLimit: 100, TrafficUsed: 100}) // over limit

	r := gin.New()
	r.GET("/api/node/vless/clients", VlessClients)

	// wrong secret -> 401
	req := httptest.NewRequest("GET", "/api/node/vless/clients", nil)
	req.Header.Set("Authorization", "wrong")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong secret should 401, got %d", w.Code)
	}

	// right secret -> only alice
	req = httptest.NewRequest("GET", "/api/node/vless/clients", nil)
	req.Header.Set("Authorization", "s3cret")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var got []map[string]string
	json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 1 || got[0]["email"] != "alice" || got[0]["uuid"] == "" || got[0]["password"] == "" {
		t.Fatalf("expected only active alice with uuid+password, got %#v", got)
	}
}
