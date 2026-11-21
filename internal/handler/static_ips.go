package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
)

type staticIPUserBrief struct {
	ID         uint      `json:"id"`
	Username   string    `json:"username"`
	ExpiresAt  time.Time `json:"expires_at"`
	Traffic24h int64     `json:"traffic_24h"`
}

type StaticIPRow struct {
	PlanID        uint                `json:"plan_id"`
	PlanName      string              `json:"plan_name"`
	ProxyType     string              `json:"proxy_type"`
	ProxyHost     string              `json:"proxy_host"`
	ProxyPort     int                 `json:"proxy_port"`
	ProxyUsername string              `json:"proxy_username"`
	ProxyNote     string              `json:"proxy_note"`
	Healthy       bool                `json:"healthy"`
	LastProbedAt  *time.Time          `json:"last_probed_at,omitempty"`
	LastRTTms     int                 `json:"last_rtt_ms"`
	LastExitIP    string              `json:"last_exit_ip"`
	Users         []staticIPUserBrief `json:"users"`
	Traffic24hTX  int64               `json:"traffic_24h_tx"`
	Traffic24hRX  int64               `json:"traffic_24h_rx"`
}

// ListStaticIPs returns the catalog of IP-bearing Plans with health info
// and bound users with their 24h traffic.
func ListStaticIPs(c *gin.Context) {
	var plans []model.Plan
	database.DB.Where("proxy_host <> ''").Order("sort_order asc, id asc").Find(&plans)

	rows := make([]StaticIPRow, 0, len(plans))

	for _, p := range plans {
		row := StaticIPRow{
			PlanID:        p.ID,
			PlanName:      p.Name,
			ProxyType:     p.ProxyType,
			ProxyHost:     p.ProxyHost,
			ProxyPort:     p.ProxyPort,
			ProxyUsername: p.ProxyUsername,
			ProxyNote:     p.ProxyNote,
		}

		if h, ok := service.GetStaticIPHealth(p.ID); ok {
			row.Healthy = h.Healthy
			t := h.LastProbedAt
			row.LastProbedAt = &t
			row.LastRTTms = h.LastRTTms
			row.LastExitIP = h.LastExitIP
		}

		var users []model.User
		database.DB.Where("plan_id = ?", p.ID).Find(&users)
		for _, u := range users {
			sums := service.TodayTrafficFor(u.Username)
			row.Users = append(row.Users, staticIPUserBrief{
				ID:         u.ID,
				Username:   u.Username,
				ExpiresAt:  u.ExpiresAt,
				Traffic24h: sums.Total(),
			})
			row.Traffic24hTX += sums.TX
			row.Traffic24hRX += sums.RX
		}

		rows = append(rows, row)
	}

	c.JSON(http.StatusOK, rows)
}
