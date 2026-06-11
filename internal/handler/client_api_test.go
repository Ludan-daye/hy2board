package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/middleware"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupClientAPITest(t *testing.T) (*gin.Engine, model.User, model.Node, model.Node) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	oldDB := database.DB
	oldConfig := config.C
	t.Cleanup(func() {
		database.DB = oldDB
		config.C = oldConfig
	})

	config.C = config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret",
			Expiry: "24h",
		},
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Node{},
		&model.Plan{},
		&model.TrafficLog{},
		&model.NodeProbeState{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db

	plan := model.Plan{
		Name:          "Pro",
		TrafficLimit:  1000,
		DurationDays:  30,
		NodeIDs:       "all",
		RuleAI:        true,
		RuleStreaming: true,
		RuleChina:     true,
		AutoReset:     true,
		PriceCents:    9900,
	}
	if err := db.Create(&plan).Error; err != nil {
		t.Fatalf("create plan: %v", err)
	}

	expires := time.Now().Add(30 * 24 * time.Hour).UTC().Truncate(time.Second)
	user := model.User{
		Username:      "alice",
		Email:         "alice@example.com",
		LoginPassword: "hashed-login-password",
		Hy2Password:   "alice:hy2-secret",
		SubToken:      "sub-secret-token",
		TrafficLimit:  1000,
		TrafficUsed:   250,
		ExpiresAt:     expires,
		Enabled:       true,
		NodeIDs:       "all",
		RuleAI:        true,
		RuleStreaming: false,
		RuleChina:     true,
		RuleAdBlock:   false,
		PlanID:        &plan.ID,
		ProxyPassword: "proxy-secret",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	healthy := model.Node{
		Name:          "JP",
		Host:          "203.0.113.10",
		Port:          443,
		Password:      "node-password",
		SNI:           "bing.com",
		Insecure:      true,
		ObfsType:      "salamander",
		ObfsPassword:  "obfs-pass",
		TrafficAPI:    "http://traffic.local",
		TrafficSecret: "traffic-secret",
		Healthy:       true,
		SortOrder:     1,
	}
	if err := db.Create(&healthy).Error; err != nil {
		t.Fatalf("create healthy node: %v", err)
	}

	unhealthy := model.Node{
		Name:          "US",
		Host:          "203.0.113.20",
		Port:          443,
		Password:      "node-password",
		SNI:           "example.com",
		TrafficSecret: "do-not-leak",
		Healthy:       false,
		SortOrder:     2,
	}
	if err := db.Create(&unhealthy).Error; err != nil {
		t.Fatalf("create unhealthy node: %v", err)
	}
	if err := db.Model(&model.Node{}).Where("id = ?", unhealthy.ID).Update("healthy", false).Error; err != nil {
		t.Fatalf("mark unhealthy node: %v", err)
	}
	unhealthy.Healthy = false

	if err := db.Create(&model.NodeProbeState{
		NodeID:        healthy.ID,
		Status:        model.NodeProbeStatusOK,
		LastLatencyMS: 42,
	}).Error; err != nil {
		t.Fatalf("create probe state: %v", err)
	}

	base := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
	logs := []model.TrafficLog{
		{SampledAt: base, NodeID: healthy.ID, Username: "alice", TX: 100, RX: 200},
		{SampledAt: base.Add(time.Hour), NodeID: healthy.ID, Username: "alice", TX: 150, RX: 280},
		{SampledAt: base.Add(time.Hour), NodeID: healthy.ID, Username: "bob", TX: 999, RX: 999},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("create traffic logs: %v", err)
	}

	r := gin.New()
	r.GET("/api/app/version", AppVersion)
	r.GET("/api/app/features", AppFeatures)
	r.GET("/api/app/downloads", ListDownloads)
	userGroup := r.Group("/api/user", middleware.UserAuth())
	{
		userGroup.GET("/session", ClientSession)
		userGroup.GET("/app/bootstrap", ClientBootstrap)
		userGroup.GET("/profile", ClientProfile)
		userGroup.GET("/plan", ClientPlan)
		userGroup.GET("/traffic/summary", ClientTrafficSummary)
		userGroup.GET("/traffic/history", ClientTrafficHistory)
		userGroup.GET("/traffic/nodes", ClientTrafficNodes)
		userGroup.GET("/nodes", ClientNodes)
		userGroup.GET("/client-config", ClientConfig)
		userGroup.GET("/announcements", ClientAnnouncements)
		userGroup.GET("/help", ClientHelp)
		userGroup.GET("/diagnostics", ClientDiagnostics)
	}

	return r, user, healthy, unhealthy
}

func userToken(t *testing.T, username string) string {
	t.Helper()
	token, err := util.GenerateUserToken(username)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return token
}

func authedGET(t *testing.T, r http.Handler, path string, username string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+userToken(t, username))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return out
}

func TestClientAPIRoutesRequireUserToken(t *testing.T) {
	r, _, _, _ := setupClientAPITest(t)
	req := httptest.NewRequest(http.MethodGet, "/api/user/profile", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
}

func TestClientBootstrapReturnsDashboardAndRedactsSecrets(t *testing.T) {
	r, _, _, _ := setupClientAPITest(t)
	rec := authedGET(t, r, "/api/user/app/bootstrap", "alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	raw := rec.Body.String()
	for _, secret := range []string{"traffic-secret", "do-not-leak", "hashed-login-password", "sub-secret-token", "proxy-secret"} {
		if strings.Contains(raw, secret) {
			t.Fatalf("response leaked %q: %s", secret, raw)
		}
	}
	profile := body["profile"].(map[string]any)
	if profile["username"] != "alice" || profile["email"] != "alice@example.com" {
		t.Fatalf("bad profile: %#v", profile)
	}
	plan := body["plan"].(map[string]any)
	if plan["name"] != "Pro" {
		t.Fatalf("bad plan: %#v", plan)
	}
	nodes := body["nodes"].([]any)
	if len(nodes) != 1 {
		t.Fatalf("nodes len = %d, want 1: %#v", len(nodes), nodes)
	}
	features := body["features"].(map[string]any)
	if features["hysteria2"] != true || features["write_account_actions"] != false {
		t.Fatalf("bad features: %#v", features)
	}
}

func TestClientNodesRespectWhitelistAndHideTransportSecrets(t *testing.T) {
	r, user, healthy, unhealthy := setupClientAPITest(t)
	if err := database.DB.Model(&model.User{}).Where("id = ?", user.ID).Update("node_ids", strconv.FormatUint(uint64(healthy.ID), 10)).Error; err != nil {
		t.Fatalf("update whitelist: %v", err)
	}
	rec := authedGET(t, r, "/api/user/nodes", "alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	raw := rec.Body.String()
	if strings.Contains(raw, "traffic-secret") || strings.Contains(raw, "traffic_api") || strings.Contains(raw, "do-not-leak") {
		t.Fatalf("node response leaked transport secret: %s", raw)
	}
	var nodes []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("decode nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("nodes len = %d, want 1: %#v", len(nodes), nodes)
	}
	if uint(nodes[0]["id"].(float64)) != healthy.ID {
		t.Fatalf("returned wrong node, unhealthy ID=%d healthy ID=%d body=%#v", unhealthy.ID, healthy.ID, nodes)
	}
	if nodes[0]["probe_status"] != model.NodeProbeStatusOK {
		t.Fatalf("missing probe status: %#v", nodes[0])
	}
}

func TestClientNodesIncludesHealthyNodeWithoutProbeState(t *testing.T) {
	r, _, healthy, _ := setupClientAPITest(t)
	noProbe := model.Node{
		Name:      "SG",
		Host:      "203.0.113.30",
		Port:      443,
		Password:  "node-password",
		SNI:       "example.org",
		Healthy:   true,
		SortOrder: 3,
	}
	if err := database.DB.Create(&noProbe).Error; err != nil {
		t.Fatalf("create healthy node without probe state: %v", err)
	}

	rec := authedGET(t, r, "/api/user/nodes", "alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}

	var nodes []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("decode nodes: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("nodes len = %d, want 2: %#v", len(nodes), nodes)
	}

	got := map[uint]bool{}
	for _, node := range nodes {
		got[uint(node["id"].(float64))] = true
	}
	if !got[healthy.ID] || !got[noProbe.ID] {
		t.Fatalf("missing healthy nodes, want IDs %d and %d: %#v", healthy.ID, noProbe.ID, nodes)
	}
}

func TestClientConfigReturnsHy2AuthAndSanitizedNodes(t *testing.T) {
	r, _, healthy, _ := setupClientAPITest(t)
	rec := authedGET(t, r, "/api/user/client-config", "alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["protocol"] != "hysteria2" || body["auth"] != "alice:hy2-secret" {
		t.Fatalf("bad config header: %#v", body)
	}
	if uint(body["default_node_id"].(float64)) != healthy.ID {
		t.Fatalf("default node mismatch: %#v", body)
	}
	raw := rec.Body.String()
	if strings.Contains(raw, "traffic-secret") || strings.Contains(raw, "traffic_api") {
		t.Fatalf("client config leaked traffic fields: %s", raw)
	}
	nodes := body["nodes"].([]any)
	if len(nodes) != 1 {
		t.Fatalf("nodes len = %d, want 1", len(nodes))
	}
	node := nodes[0].(map[string]any)
	if node["obfs_password"] != "obfs-pass" || node["sni"] != "bing.com" {
		t.Fatalf("missing hy2 fields: %#v", node)
	}
}

func TestClientTrafficHistoryOnlyReturnsCurrentUser(t *testing.T) {
	r, _, _, _ := setupClientAPITest(t)
	rec := authedGET(t, r, "/api/user/traffic/history?days=7", "alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	raw := rec.Body.String()
	if strings.Contains(raw, "bob") {
		t.Fatalf("history leaked another user: %s", raw)
	}
	var rows []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows len = %d, want 2: %#v", len(rows), rows)
	}
}

func TestClientTrafficHistoryUsesClientDTO(t *testing.T) {
	r, _, _, _ := setupClientAPITest(t)
	rec := authedGET(t, r, "/api/user/traffic/history?days=7", "alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}

	var rows []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode rows: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected traffic history rows: %s", rec.Body.String())
	}

	row := rows[0]
	for _, field := range []string{"sampled_at", "node_id", "tx", "rx"} {
		if _, ok := row[field]; !ok {
			t.Fatalf("history row missing %q: %#v", field, row)
		}
	}
	for _, field := range []string{"id", "username"} {
		if _, ok := row[field]; ok {
			t.Fatalf("history row exposed raw model field %q: %#v", field, row)
		}
	}
}

func TestClientDiagnosticsReportsAccountAndNodeState(t *testing.T) {
	r, user, _, _ := setupClientAPITest(t)
	if err := database.DB.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"traffic_used":  int64(1200),
		"traffic_limit": int64(1000),
	}).Error; err != nil {
		t.Fatalf("update user: %v", err)
	}
	rec := authedGET(t, r, "/api/user/diagnostics", "alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	reasons := body["reason_codes"].([]any)
	found := false
	for _, reason := range reasons {
		if reason == "traffic_exceeded" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing traffic_exceeded reason: %#v", body)
	}
	if body["can_connect"] != false {
		t.Fatalf("expected can_connect=false: %#v", body)
	}
}

func TestClientDiagnosticsAllowsConnectionWithPartialNodeOutage(t *testing.T) {
	r, _, _, _ := setupClientAPITest(t)
	rec := authedGET(t, r, "/api/user/diagnostics", "alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}

	body := decodeBody(t, rec)
	if body["can_connect"] != true {
		t.Fatalf("expected can_connect=true with at least one healthy node: %#v", body)
	}

	reasons := body["reason_codes"].([]any)
	for _, reason := range reasons {
		switch reason {
		case "account_disabled", "plan_expired", "traffic_exceeded", "no_nodes":
			t.Fatalf("unexpected account blocker %q: %#v", reason, body)
		}
	}
}
