package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
)

type NodeTrafficInfo struct {
	ID      uint                           `json:"id"`
	Name    string                         `json:"name"`
	Host    string                         `json:"host"`
	Port    int                            `json:"port"`
	Healthy bool                           `json:"healthy"`
	Online  int                            `json:"online"`
	Traffic map[string]service.TrafficData `json:"traffic"`
}

func GetStats(c *gin.Context) {
	var totalUsers int64
	var activeUsers int64
	var totalNodes int64
	var healthyNodes int64

	database.DB.Model(&model.User{}).Count(&totalUsers)
	database.DB.Model(&model.User{}).Where("enabled = ?", true).Count(&activeUsers)
	database.DB.Model(&model.Node{}).Count(&totalNodes)
	database.DB.Model(&model.Node{}).Where("healthy = ?", true).Count(&healthyNodes)

	snaps := service.GetTrafficSnapshot()
	results := make([]NodeTrafficInfo, len(snaps))
	var totalTX, totalRX int64
	for i, s := range snaps {
		results[i] = NodeTrafficInfo{
			ID:      s.ID,
			Name:    s.Name,
			Host:    s.Host,
			Port:    s.Port,
			Healthy: s.Healthy,
			Online:  s.Online,
			Traffic: s.Traffic,
		}
		for _, t := range s.Traffic {
			totalTX += t.TX
			totalRX += t.RX
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_users":   totalUsers,
		"active_users":  activeUsers,
		"total_nodes":   totalNodes,
		"healthy_nodes": healthyNodes,
		"total_tx":      totalTX,
		"total_rx":      totalRX,
		"nodes":         results,
		"history":       service.GetHistory(),
	})
}

// GetHistory returns the rolling traffic time-series. Instant — all computed in background.
func GetHistory(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"history": service.GetHistory(),
	})
}

func GetBuckets(c *gin.Context) {
	c.JSON(http.StatusOK, service.GetUserBuckets())
}
