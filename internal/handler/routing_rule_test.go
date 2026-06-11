package handler

import (
	"bytes"
	"encoding/json"
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

func TestRoutingRuleCRUDAndPreview(t *testing.T) {
	setupRoutingRuleHandlerTestDB(t)
	router := routingRuleTestRouter()

	createBody := map[string]interface{}{
		"enabled":    true,
		"name":       "Scholar",
		"kind":       "DOMAIN-SUFFIX",
		"value":      "scholar.google.com",
		"policy":     "HongKong",
		"sort_order": 10,
		"note":       "force scholar via HK",
	}
	create := performJSON(router, http.MethodPost, "/routing-rules", createBody)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", create.Code, create.Body.String())
	}
	var created model.CustomRoutingRule
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || !created.Enabled || created.Policy != "HongKong" {
		t.Fatalf("unexpected created rule: %+v", created)
	}

	list := performJSON(router, http.MethodGet, "/routing-rules", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", list.Code, list.Body.String())
	}
	if !strings.Contains(list.Body.String(), "scholar.google.com") {
		t.Fatalf("list missing created rule: %s", list.Body.String())
	}

	updateBody := map[string]interface{}{
		"enabled":    false,
		"name":       "Scholar Disabled",
		"kind":       "DOMAIN-SUFFIX",
		"value":      "scholar.google.com",
		"policy":     "Auto",
		"sort_order": 20,
		"note":       "disabled",
	}
	update := performJSON(router, http.MethodPut, "/routing-rules/1", updateBody)
	if update.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", update.Code, update.Body.String())
	}
	if !strings.Contains(update.Body.String(), `"enabled":false`) || !strings.Contains(update.Body.String(), `"policy":"Auto"`) {
		t.Fatalf("update response mismatch: %s", update.Body.String())
	}

	preview := performJSON(router, http.MethodPost, "/routing-rules/preview", createBody)
	if preview.Code != http.StatusOK {
		t.Fatalf("preview status = %d body=%s", preview.Code, preview.Body.String())
	}
	for _, want := range []string{
		`"surge":"DOMAIN-SUFFIX,scholar.google.com,Auto"`,
		`"shadowrocket":"DOMAIN-SUFFIX,scholar.google.com,Auto"`,
		`"clash":"  - DOMAIN-SUFFIX,scholar.google.com,Auto"`,
	} {
		if !strings.Contains(preview.Body.String(), want) {
			t.Fatalf("preview missing %s: %s", want, preview.Body.String())
		}
	}

	del := performJSON(router, http.MethodDelete, "/routing-rules/1", nil)
	if del.Code != http.StatusOK {
		t.Fatalf("delete status = %d body=%s", del.Code, del.Body.String())
	}
	afterDelete := performJSON(router, http.MethodGet, "/routing-rules", nil)
	if strings.Contains(afterDelete.Body.String(), "scholar.google.com") {
		t.Fatalf("deleted rule still listed: %s", afterDelete.Body.String())
	}
}

func TestRoutingRuleRejectsInvalidInput(t *testing.T) {
	setupRoutingRuleHandlerTestDB(t)
	router := routingRuleTestRouter()

	cases := []map[string]interface{}{
		{"name": "Bad kind", "kind": "RULE-SET", "value": "example.com", "policy": "Auto"},
		{"name": "Bad policy", "kind": "DOMAIN-SUFFIX", "value": "example.com", "policy": "MissingGroup"},
		{"name": "Bad cidr", "kind": "IP-CIDR", "value": "not-cidr", "policy": "DIRECT"},
		{"name": "Bad domain", "kind": "DOMAIN-SUFFIX", "value": "bad,domain", "policy": "Auto"},
	}

	for _, body := range cases {
		res := performJSON(router, http.MethodPost, "/routing-rules", body)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %+v, got %d body=%s", body, res.Code, res.Body.String())
		}
	}
}

func setupRoutingRuleHandlerTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.CustomRoutingRule{}, &model.AuditLog{}, &model.Node{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db
}

func routingRuleTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/routing-rules", ListRoutingRules)
	r.POST("/routing-rules", CreateRoutingRule)
	r.PUT("/routing-rules/:id", UpdateRoutingRule)
	r.DELETE("/routing-rules/:id", DeleteRoutingRule)
	r.POST("/routing-rules/preview", PreviewRoutingRule)
	return r
}

func performJSON(r http.Handler, method string, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}
