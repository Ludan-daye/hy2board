package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestListUsersAggregatesTodayTrafficForAllUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := database.DB
	t.Cleanup(func() {
		database.DB = previousDB
		service.ClearTodayTrafficCache()
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Node{}, &model.TrafficLog{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	database.DB = db

	users := []model.User{
		{Username: "alice", Hy2Password: "alice-password", SubToken: "alice-token", Enabled: true},
		{Username: "bob", Hy2Password: "bob-password", SubToken: "bob-token", Enabled: true},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Create(&model.Node{Name: "node", Host: "node.example", Port: 443, Healthy: true}).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	recent := time.Now().Add(-time.Hour)
	old := time.Now().Add(-48 * time.Hour)
	logs := []model.TrafficLog{
		{SampledAt: old, NodeID: 1, Username: "alice", TX: 1, RX: 1},
		{SampledAt: recent, NodeID: 1, Username: "alice", TX: 10, RX: 5},
		{SampledAt: recent.Add(time.Minute), NodeID: 1, Username: "alice", TX: 20, RX: 15},
		{SampledAt: recent, NodeID: 2, Username: "alice", TX: 100, RX: 50},
		{SampledAt: recent.Add(time.Minute), NodeID: 2, Username: "alice", TX: 130, RX: 70},
		{SampledAt: recent, NodeID: 1, Username: "bob", TX: 1, RX: 2},
		{SampledAt: recent.Add(time.Minute), NodeID: 1, Username: "bob", TX: 4, RX: 5},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("create traffic logs: %v", err)
	}

	if err := service.ReloadTodayTraffic(); err != nil {
		t.Fatalf("reload today traffic: %v", err)
	}

	router := gin.New()
	router.GET("/users", ListUsers)
	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}

	var rows []userListRow
	if err := json.Unmarshal(response.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	totals := make(map[string]int64, len(rows))
	for _, row := range rows {
		totals[row.Username] = row.TodayTraffic
	}
	if totals["alice"] != 70 {
		t.Fatalf("alice today traffic = %d, want 70", totals["alice"])
	}
	if totals["bob"] != 6 {
		t.Fatalf("bob today traffic = %d, want 6", totals["bob"])
	}
}
