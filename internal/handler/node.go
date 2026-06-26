package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

type NodeRequest struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Password      string `json:"password"`
	SNI           string `json:"sni"`
	Insecure      *bool  `json:"insecure"`
	ObfsType      string `json:"obfs_type"`
	ObfsPassword  string `json:"obfs_password"`
	TrafficAPI    string `json:"traffic_api"`
	TrafficSecret string `json:"traffic_secret"`
	SortOrder     int    `json:"sort_order"`

	VlessEnabled     *bool  `json:"vless_enabled"`
	VlessPort        int    `json:"vless_port"`
	RealityPubkey    string `json:"reality_pubkey"`
	RealityShortID   string `json:"reality_shortid"`
	RealitySNI       string `json:"reality_sni"`
	VlessStatsAPI    string `json:"vless_stats_api"`
	VlessStatsSecret string `json:"vless_stats_secret"`
	TrojanEnabled    *bool  `json:"trojan_enabled"`
	TrojanPort       int    `json:"trojan_port"`
	TrojanSNI        string `json:"trojan_sni"`
}

type NodeProbeView struct {
	Status        string     `json:"status"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	LastOKAt      *time.Time `json:"last_ok_at,omitempty"`
	LastFailedAt  *time.Time `json:"last_failed_at,omitempty"`
	LastLatencyMS int        `json:"last_latency_ms"`
	LastError     string     `json:"last_error"`
	FailStreak    int        `json:"fail_streak"`
	AlertedDown   bool       `json:"alerted_down"`
}

type NodeRow struct {
	model.Node
	Probe *NodeProbeView `json:"probe,omitempty"`
}

func ListNodes(c *gin.Context) {
	var nodes []model.Node
	database.DB.Order("sort_order asc").Find(&nodes)

	var states []model.NodeProbeState
	database.DB.Find(&states)
	stateByNode := make(map[uint]model.NodeProbeState, len(states))
	for _, s := range states {
		stateByNode[s.NodeID] = s
	}

	rows := make([]NodeRow, 0, len(nodes))
	for _, n := range nodes {
		row := NodeRow{Node: n}
		if s, ok := stateByNode[n.ID]; ok {
			row.Probe = &NodeProbeView{
				Status:        s.Status,
				LastCheckedAt: s.LastCheckedAt,
				LastOKAt:      s.LastOKAt,
				LastFailedAt:  s.LastFailedAt,
				LastLatencyMS: s.LastLatencyMS,
				LastError:     s.LastError,
				FailStreak:    s.FailStreak,
				AlertedDown:   s.AlertedDown,
			}
		}
		rows = append(rows, row)
	}
	c.JSON(http.StatusOK, rows)
}

func CreateNode(c *gin.Context) {
	var req NodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	node := model.Node{
		Name:          req.Name,
		Host:          req.Host,
		Port:          req.Port,
		Password:      req.Password,
		SNI:           req.SNI,
		Insecure:      req.Insecure != nil && *req.Insecure,
		ObfsType:      req.ObfsType,
		ObfsPassword:  req.ObfsPassword,
		TrafficAPI:    req.TrafficAPI,
		TrafficSecret: req.TrafficSecret,
		SortOrder:     req.SortOrder,
		Healthy:       true,

		VlessEnabled:     req.VlessEnabled != nil && *req.VlessEnabled,
		VlessPort:        req.VlessPort,
		RealityPubkey:    req.RealityPubkey,
		RealityShortID:   req.RealityShortID,
		RealitySNI:       req.RealitySNI,
		VlessStatsAPI:    req.VlessStatsAPI,
		VlessStatsSecret: req.VlessStatsSecret,
		TrojanEnabled:    req.TrojanEnabled != nil && *req.TrojanEnabled,
		TrojanPort:       req.TrojanPort,
		TrojanSNI:        req.TrojanSNI,
	}
	database.DB.Create(&node)
	auditAdminAction(c, "node.create", "node", node.ID, node.Name, map[string]interface{}{
		"host": node.Host,
		"port": node.Port,
		"sni":  node.SNI,
		"obfs": node.ObfsType,
	})
	c.JSON(http.StatusCreated, node)
}

func UpdateNode(c *gin.Context) {
	var node model.Node
	if database.DB.First(&node, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	var req NodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"name":           req.Name,
		"host":           req.Host,
		"port":           req.Port,
		"password":       req.Password,
		"sni":            req.SNI,
		"obfs_type":      req.ObfsType,
		"obfs_password":  req.ObfsPassword,
		"traffic_api":    req.TrafficAPI,
		"traffic_secret": req.TrafficSecret,
		"sort_order":     req.SortOrder,
	}
	if req.Insecure != nil {
		updates["insecure"] = *req.Insecure
	}

	updates["vless_port"] = req.VlessPort
	updates["reality_pubkey"] = req.RealityPubkey
	updates["reality_short_id"] = req.RealityShortID // DB column is reality_short_id
	updates["reality_sni"] = req.RealitySNI
	updates["vless_stats_api"] = req.VlessStatsAPI
	updates["trojan_port"] = req.TrojanPort
	updates["trojan_sni"] = req.TrojanSNI
	if req.VlessEnabled != nil {
		updates["vless_enabled"] = *req.VlessEnabled
	}
	if req.TrojanEnabled != nil {
		updates["trojan_enabled"] = *req.TrojanEnabled
	}
	if req.VlessStatsSecret != "" { // write-only: blank leaves existing unchanged
		updates["vless_stats_secret"] = req.VlessStatsSecret
	}

	database.DB.Model(&node).Updates(updates)
	database.DB.First(&node, node.ID)
	auditAdminAction(c, "node.update", "node", node.ID, node.Name, map[string]interface{}{"fields": auditKeys(updates)})
	c.JSON(http.StatusOK, node)
}

func DeleteNode(c *gin.Context) {
	var node model.Node
	if database.DB.First(&node, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	database.DB.Unscoped().Delete(&node)
	auditAdminAction(c, "node.delete", "node", node.ID, node.Name, map[string]interface{}{
		"host": node.Host,
		"port": node.Port,
	})
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
