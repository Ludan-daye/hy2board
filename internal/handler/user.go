package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
	"gorm.io/gorm"
)

var subTokenRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,64}$`)
var nodeIDsRegex = regexp.MustCompile(`^(all|\d+(,\d+)*)$`)

func validateSubToken(token string, excludeUserID uint) error {
	if !subTokenRegex.MatchString(token) {
		return fmt.Errorf("sub_token must match ^[a-zA-Z0-9_-]{8,64}$")
	}
	var count int64
	database.DB.Model(&model.User{}).
		Where("sub_token = ? AND id <> ?", token, excludeUserID).
		Count(&count)
	if count > 0 {
		return fmt.Errorf("sub_token already in use")
	}
	return nil
}

func validateHy2Password(p string) error {
	if len(p) < 8 {
		return fmt.Errorf("hy2_password must be at least 8 characters")
	}
	return nil
}

func validateNodeIDs(s string) error {
	if !nodeIDsRegex.MatchString(s) {
		return fmt.Errorf("node_ids must be 'all' or comma-separated digits")
	}
	return nil
}

type CreateUserRequest struct {
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password"`
	TrafficLimit int64  `json:"traffic_limit"`
	MaxIPs       int    `json:"max_ips"`
	ExpiresAt    string `json:"expires_at"`
	Enabled      *bool  `json:"enabled"`
	ChainProxy   *bool  `json:"chain_proxy"`
}

type UpdateUserRequest struct {
	// Existing
	Username     *string `json:"username"`
	TrafficLimit *int64  `json:"traffic_limit"`
	MaxIPs       *int    `json:"max_ips"`
	ExpiresAt    string  `json:"expires_at"`
	Enabled      *bool   `json:"enabled"`
	ChainProxy   *bool   `json:"chain_proxy"`

	// Subscription
	Hy2Password *string `json:"hy2_password"`
	SubToken    *string `json:"sub_token"`
	NodeIDs     *string `json:"node_ids"`

	// Meta
	Email *string `json:"email"`
	Notes *string `json:"notes"`
	Tags  *string `json:"tags"`

	// Rule flags (written, but subscribe handler won't honor them until Phase 2)
	RuleAI        *bool `json:"rule_ai"`
	RuleStreaming *bool `json:"rule_streaming"`
	RuleChina     *bool `json:"rule_china"`
	RuleAdBlock   *bool `json:"rule_ad_block"`
	AutoReset     *bool `json:"auto_reset"`
	PlanID        *uint `json:"plan_id"`

	// Telegram
	TelegramID *int64 `json:"telegram_id"`

	// Static IP proxy override
	ProxyType     *string `json:"proxy_type"`
	ProxyHost     *string `json:"proxy_host"`
	ProxyPort     *int    `json:"proxy_port"`
	ProxyUsername *string `json:"proxy_username"`
}

type userListRow struct {
	model.User
	SpeedTX       int64      `json:"speed_tx"`
	SpeedRX       int64      `json:"speed_rx"`
	LiveTX        int64      `json:"live_tx"`
	LiveRX        int64      `json:"live_rx"`
	ActiveNodes   int        `json:"active_nodes"`
	AssignedNodes int        `json:"assigned_nodes"`
	NextResetAt   *time.Time `json:"next_reset_at,omitempty"`
	TodayTraffic  int64      `json:"today_traffic"`
}

func ListUsers(c *gin.Context) {
	var users []model.User
	database.DB.Find(&users)

	var healthyNodes []model.Node
	database.DB.Where("healthy = ?", true).Find(&healthyNodes)
	healthyNodeCount := len(healthyNodes)

	out := make([]userListRow, 0, len(users))
	for _, u := range users {
		row := userListRow{User: u}
		if s, ok := service.GetUserStat(u.Username); ok {
			row.SpeedTX = s.SpeedTX
			row.SpeedRX = s.SpeedRX
			row.LiveTX = s.TotalTX
			row.LiveRX = s.TotalRX
		}

		// ActiveNodes: count nodes actively carrying traffic for this user (from cache)
		if s, ok := service.GetUserStat(u.Username); ok {
			for _, nt := range s.PerNode {
				if nt.TX > 0 || nt.RX > 0 {
					row.ActiveNodes++
				}
			}
		}

		// AssignedNodes: node whitelist intersected with healthy nodes
		if allowed := u.EffectiveNodeIDs(); allowed == nil {
			row.AssignedNodes = healthyNodeCount
		} else {
			for _, n := range healthyNodes {
				if allowed[n.ID] {
					row.AssignedNodes++
				}
			}
		}

		// NextResetAt: only if auto_reset is on
		if u.AutoReset && !u.LastResetAt.IsZero() {
			next := u.LastResetAt.Add(30 * 24 * time.Hour)
			row.NextResetAt = &next
		}

		// TodayTraffic: 24h delta from cumulative counter snapshots
		// (per-node MAX-MIN, summed across nodes; raw SUM would double-count).
		var lastDay struct {
			SumTX int64
			SumRX int64
		}
		database.DB.Raw(`
			SELECT COALESCE(SUM(dtx),0) AS sum_tx, COALESCE(SUM(drx),0) AS sum_rx FROM (
				SELECT MAX(tx)-MIN(tx) AS dtx, MAX(rx)-MIN(rx) AS drx
				FROM traffic_logs
				WHERE username = ? AND sampled_at >= ?
				GROUP BY node_id
			)`, u.Username, time.Now().Add(-24*time.Hour)).Scan(&lastDay)
		row.TodayTraffic = lastDay.SumTX + lastDay.SumRX

		out = append(out, row)
	}
	c.JSON(http.StatusOK, out)
}

func GetUser(c *gin.Context) {
	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := model.User{
		Username:     req.Username,
		TrafficLimit: req.TrafficLimit,
		MaxIPs:       req.MaxIPs,
		Enabled:      true,
	}

	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}
	if req.ChainProxy != nil {
		user.ChainProxy = *req.ChainProxy
	}

	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			t, err = time.Parse("2006-01-02", req.ExpiresAt)
		}
		if err == nil {
			user.ExpiresAt = t
		}
	}

	if req.Password != "" {
		user.SetLoginPassword(req.Password)
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	auditAdminAction(c, "user.create", "user", user.ID, user.Username, map[string]interface{}{
		"enabled":       user.Enabled,
		"traffic_limit": user.TrafficLimit,
		"chain_proxy":   user.ChainProxy,
	})

	c.JSON(http.StatusCreated, user)
}

func UpdateUser(c *gin.Context) {
	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}

	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.TrafficLimit != nil {
		updates["traffic_limit"] = *req.TrafficLimit
	}
	if req.MaxIPs != nil {
		updates["max_ips"] = *req.MaxIPs
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.ChainProxy != nil {
		updates["chain_proxy"] = *req.ChainProxy
	}
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			t, err = time.Parse("2006-01-02", req.ExpiresAt)
		}
		if err == nil {
			updates["expires_at"] = t
		}
	}

	if req.Hy2Password != nil {
		if err := validateHy2Password(*req.Hy2Password); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates["hy2_password"] = *req.Hy2Password
	}
	if req.SubToken != nil {
		if err := validateSubToken(*req.SubToken, user.ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates["sub_token"] = *req.SubToken
	}
	if req.NodeIDs != nil {
		if err := validateNodeIDs(*req.NodeIDs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		updates["node_ids"] = *req.NodeIDs
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Notes != nil {
		updates["notes"] = *req.Notes
	}
	if req.Tags != nil {
		seen := map[string]bool{}
		var out []string
		for _, t := range strings.Split(*req.Tags, ",") {
			t = strings.TrimSpace(t)
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			out = append(out, t)
		}
		updates["tags"] = strings.Join(out, ",")
	}

	// Mirror chain_proxy <-> rule_ai for back-compat
	if req.ChainProxy != nil && req.RuleAI == nil {
		updates["rule_ai"] = *req.ChainProxy
	}
	if req.RuleAI != nil && req.ChainProxy == nil {
		updates["chain_proxy"] = *req.RuleAI
		updates["rule_ai"] = *req.RuleAI
	}

	if req.RuleStreaming != nil {
		updates["rule_streaming"] = *req.RuleStreaming
	}
	if req.RuleChina != nil {
		updates["rule_china"] = *req.RuleChina
	}
	if req.RuleAdBlock != nil {
		updates["rule_ad_block"] = *req.RuleAdBlock
	}
	if req.AutoReset != nil {
		updates["auto_reset"] = *req.AutoReset
	}
	if req.PlanID != nil {
		updates["plan_id"] = *req.PlanID
	}
	if req.TelegramID != nil {
		updates["telegram_id"] = *req.TelegramID
	}
	if req.ProxyType != nil {
		updates["proxy_type"] = *req.ProxyType
	}
	if req.ProxyHost != nil {
		updates["proxy_host"] = *req.ProxyHost
	}
	if req.ProxyPort != nil {
		updates["proxy_port"] = *req.ProxyPort
	}
	if req.ProxyUsername != nil {
		updates["proxy_username"] = *req.ProxyUsername
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.First(&user, user.ID)
	auditAdminAction(c, "user.update", "user", user.ID, user.Username, map[string]interface{}{"fields": auditKeys(updates)})
	c.JSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context) {
	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	database.DB.Unscoped().Delete(&user)
	auditAdminAction(c, "user.delete", "user", user.ID, user.Username, nil)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func ResetSubToken(c *gin.Context) {
	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	b := make([]byte, 16)
	rand.Read(b)
	newToken := hex.EncodeToString(b)
	database.DB.Model(&user).Update("sub_token", newToken)
	database.DB.First(&user, user.ID)
	auditAdminAction(c, "user.reset_sub_token", "user", user.ID, user.Username, nil)
	c.JSON(http.StatusOK, user)
}

func ToggleUser(c *gin.Context) {
	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	database.DB.Model(&user).Update("enabled", !user.Enabled)
	database.DB.First(&user, user.ID)
	auditAdminAction(c, "user.toggle", "user", user.ID, user.Username, map[string]interface{}{"enabled": user.Enabled})
	c.JSON(http.StatusOK, user)
}

func ResetTraffic(c *gin.Context) {
	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	database.DB.Model(&user).Update("traffic_used", 0)
	auditAdminAction(c, "user.reset_traffic", "user", user.ID, user.Username, nil)
	c.JSON(http.StatusOK, gin.H{"message": "traffic reset"})
}

func ToggleChainProxy(c *gin.Context) {
	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	newVal := !user.ChainProxy
	if err := database.DB.Model(&user).Updates(map[string]interface{}{
		"chain_proxy": newVal,
		"rule_ai":     newVal,
	}).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.First(&user, user.ID)
	auditAdminAction(c, "user.toggle_chain_proxy", "user", user.ID, user.Username, map[string]interface{}{"rule_ai": user.RuleAI})
	c.JSON(http.StatusOK, user)
}

type RenewRequest struct {
	Days    int           `json:"days" binding:"required"`
	Payment *PaymentInput `json:"payment,omitempty"`
}

func RenewUser(c *gin.Context) {
	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	var req RenewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Days <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "days must be > 0"})
		return
	}
	base := time.Now()
	if user.ExpiresAt.After(base) {
		base = user.ExpiresAt
	}
	newExpiry := base.AddDate(0, 0, req.Days)

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("expires_at", newExpiry).Error; err != nil {
			return err
		}
		if req.Payment != nil {
			var planID uint
			if user.PlanID != nil {
				planID = *user.PlanID
			}
			return CreatePayment(tx, user.ID, planID, req.Days, c.GetString("admin"), *req.Payment)
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	database.DB.First(&user, user.ID)
	auditAdminAction(c, "user.renew", "user", user.ID, user.Username, map[string]interface{}{
		"days":        req.Days,
		"has_payment": req.Payment != nil,
	})
	c.JSON(http.StatusOK, user)
}

type BulkRequest struct {
	IDs    []uint `json:"ids" binding:"required"`
	Action string `json:"action" binding:"required"`
}

func BulkUsers(c *gin.Context) {
	var req BulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids must be non-empty"})
		return
	}
	switch req.Action {
	case "enable":
		database.DB.Model(&model.User{}).Where("id IN ?", req.IDs).Update("enabled", true)
	case "disable":
		database.DB.Model(&model.User{}).Where("id IN ?", req.IDs).Update("enabled", false)
	case "delete":
		database.DB.Unscoped().Where("id IN ?", req.IDs).Delete(&model.User{})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown action"})
		return
	}
	auditAdminAction(c, "user.bulk_"+req.Action, "user", 0, "bulk", map[string]interface{}{
		"ids":   req.IDs,
		"count": len(req.IDs),
	})
	c.JSON(http.StatusOK, gin.H{"message": "ok", "count": len(req.IDs)})
}
