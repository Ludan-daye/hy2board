package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUpdateNodePersistsVlessTrojanFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.Node{}, &model.AuditLog{})
	database.DB = db
	n := model.Node{Name: "HK1", Host: "1.2.3.4", Port: 443, VlessStatsSecret: "oldsecret"}
	db.Create(&n)

	r := gin.New()
	r.PUT("/admin/nodes/:id", UpdateNode)
	put := func(body string) {
		req := httptest.NewRequest("PUT", "/admin/nodes/"+fmt.Sprint(n.ID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("PUT got %d: %s", w.Code, w.Body.String())
		}
	}

	put(`{"name":"HK1","host":"1.2.3.4","port":443,
	  "vless_enabled":true,"vless_port":443,"reality_pubkey":"PUB","reality_shortid":"ab12",
	  "reality_sni":"www.apple.com","vless_stats_api":"http://1.2.3.4:25415/","vless_stats_secret":"",
	  "trojan_enabled":true,"trojan_port":8443,"trojan_sni":"www.apple.com"}`)

	var got model.Node
	db.First(&got, n.ID)
	if !got.VlessEnabled || got.VlessPort != 443 || got.RealityPubkey != "PUB" || got.RealityShortID != "ab12" ||
		got.RealitySNI != "www.apple.com" || got.VlessStatsAPI != "http://1.2.3.4:25415/" ||
		!got.TrojanEnabled || got.TrojanPort != 8443 || got.TrojanSNI != "www.apple.com" {
		t.Fatalf("fields not persisted: %#v", got)
	}
	if got.VlessStatsSecret != "oldsecret" {
		t.Fatalf("blank secret must leave existing unchanged, got %q", got.VlessStatsSecret)
	}

	// non-empty secret replaces; disabling persists false
	put(`{"name":"HK1","host":"1.2.3.4","port":443,"vless_enabled":false,"vless_stats_secret":"news3cret","trojan_enabled":false}`)
	db.First(&got, n.ID)
	if got.VlessEnabled || got.TrojanEnabled {
		t.Fatalf("disable must persist false")
	}
	if got.VlessStatsSecret != "news3cret" {
		t.Fatalf("non-empty secret must replace, got %q", got.VlessStatsSecret)
	}
}
