package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
	"gorm.io/gorm"
)

type CreatePlanRequest struct {
	Name          string `json:"name" binding:"required"`
	TrafficLimit  int64  `json:"traffic_limit"`
	DurationDays  int    `json:"duration_days"`
	NodeIDs       string `json:"node_ids"`
	RuleAI        bool   `json:"rule_ai"`
	RuleStreaming bool   `json:"rule_streaming"`
	RuleChina     *bool  `json:"rule_china"`
	RuleAdBlock   bool   `json:"rule_ad_block"`
	AutoReset     bool   `json:"auto_reset"`
	SortOrder     int    `json:"sort_order"`
	ProxyType     string `json:"proxy_type"`
	ProxyHost     string `json:"proxy_host"`
	ProxyPort     int    `json:"proxy_port"`
	ProxyUsername string `json:"proxy_username"`
	ProxyPassword string `json:"proxy_password"`
	ProxyNote     string `json:"proxy_note"`
	PriceCents    int64  `json:"price_cents"`
}

type UpdatePlanRequest struct {
	Name          *string `json:"name"`
	TrafficLimit  *int64  `json:"traffic_limit"`
	DurationDays  *int    `json:"duration_days"`
	NodeIDs       *string `json:"node_ids"`
	RuleAI        *bool   `json:"rule_ai"`
	RuleStreaming *bool   `json:"rule_streaming"`
	RuleChina     *bool   `json:"rule_china"`
	RuleAdBlock   *bool   `json:"rule_ad_block"`
	AutoReset     *bool   `json:"auto_reset"`
	SortOrder     *int    `json:"sort_order"`
	ProxyType     *string `json:"proxy_type"`
	ProxyHost     *string `json:"proxy_host"`
	ProxyPort     *int    `json:"proxy_port"`
	ProxyUsername *string `json:"proxy_username"`
	ProxyNote     *string `json:"proxy_note"`
	PriceCents    *int64  `json:"price_cents,omitempty"`
}

type planListRow struct {
	model.Plan
	UsersCount int64 `json:"users_count"`
}

func planUserSubscriptionUpdates(p model.Plan) map[string]interface{} {
	return map[string]interface{}{
		"traffic_limit":  p.TrafficLimit,
		"max_ips":        p.MaxIPs,
		"node_ids":       p.NodeIDs,
		"rule_ai":        p.RuleAI,
		"rule_streaming": p.RuleStreaming,
		"rule_china":     p.RuleChina,
		"rule_ad_block":  p.RuleAdBlock,
		"auto_reset":     p.AutoReset,
		"proxy_type":     p.ProxyType,
		"proxy_host":     p.ProxyHost,
		"proxy_port":     p.ProxyPort,
		"proxy_username": p.ProxyUsername,
		"proxy_password": p.ProxyPassword,
	}
}

func syncPlanBoundUsers(tx *gorm.DB, p model.Plan) (int64, error) {
	if p.ID == 0 {
		return 0, nil
	}
	result := tx.Model(&model.User{}).
		Where("plan_id = ?", p.ID).
		Updates(planUserSubscriptionUpdates(p))
	return result.RowsAffected, result.Error
}

func ListPlans(c *gin.Context) {
	var plans []model.Plan
	database.DB.Order("sort_order asc, id asc").Find(&plans)
	out := make([]planListRow, 0, len(plans))
	for _, p := range plans {
		var n int64
		database.DB.Model(&model.User{}).Where("plan_id = ?", p.ID).Count(&n)
		out = append(out, planListRow{Plan: p, UsersCount: n})
	}
	c.JSON(http.StatusOK, out)
}

func GetPlan(c *gin.Context) {
	var p model.Plan
	if database.DB.First(&p, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func CreatePlan(c *gin.Context) {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p := model.Plan{
		Name:          req.Name,
		TrafficLimit:  req.TrafficLimit,
		DurationDays:  req.DurationDays,
		NodeIDs:       req.NodeIDs,
		RuleAI:        req.RuleAI,
		RuleStreaming: req.RuleStreaming,
		RuleAdBlock:   req.RuleAdBlock,
		AutoReset:     req.AutoReset,
		SortOrder:     req.SortOrder,
		ProxyType:     req.ProxyType,
		ProxyHost:     req.ProxyHost,
		ProxyPort:     req.ProxyPort,
		ProxyUsername: req.ProxyUsername,
		ProxyPassword: req.ProxyPassword,
		ProxyNote:     req.ProxyNote,
		PriceCents:    req.PriceCents,
	}
	if req.NodeIDs == "" {
		p.NodeIDs = "all"
	}
	if req.RuleChina == nil {
		p.RuleChina = true
	} else {
		p.RuleChina = *req.RuleChina
	}
	if err := database.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if p.ProxyHost != "" {
		service.ProbeOnce(p.ID)
	}
	auditAdminAction(c, "plan.create", "plan", p.ID, p.Name, map[string]interface{}{
		"duration_days": p.DurationDays,
		"traffic_limit": p.TrafficLimit,
		"proxy_host":    p.ProxyHost,
	})
	c.JSON(http.StatusCreated, p)
}

func UpdatePlan(c *gin.Context) {
	var p model.Plan
	if database.DB.First(&p, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.TrafficLimit != nil {
		updates["traffic_limit"] = *req.TrafficLimit
	}
	if req.DurationDays != nil {
		updates["duration_days"] = *req.DurationDays
	}
	if req.NodeIDs != nil {
		updates["node_ids"] = *req.NodeIDs
	}
	if req.RuleAI != nil {
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
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
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
	if req.ProxyNote != nil {
		updates["proxy_note"] = *req.ProxyNote
	}
	if req.PriceCents != nil {
		updates["price_cents"] = *req.PriceCents
	}
	var syncedUsers int64
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&p).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.First(&p, p.ID).Error; err != nil {
			return err
		}
		var err error
		syncedUsers, err = syncPlanBoundUsers(tx, p)
		return err
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if p.ProxyHost != "" {
		service.ProbeOnce(p.ID)
	}
	auditAdminAction(c, "plan.update", "plan", p.ID, p.Name, map[string]interface{}{
		"fields":       auditKeys(updates),
		"synced_users": syncedUsers,
	})
	c.JSON(http.StatusOK, p)
}

func DeletePlan(c *gin.Context) {
	id := c.Param("id")
	var count int64
	database.DB.Model(&model.User{}).Where("plan_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":       "plan is in use",
			"users_count": count,
		})
		return
	}
	var p model.Plan
	if database.DB.First(&p, id).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	database.DB.Delete(&p)
	auditAdminAction(c, "plan.delete", "plan", p.ID, p.Name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

type ApplyPlanRequest struct {
	Payment *PaymentInput `json:"payment,omitempty"`
}

func ApplyPlanToUser(c *gin.Context) {
	var p model.Plan
	if database.DB.First(&p, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	var u model.User
	if database.DB.First(&u, c.Param("userId")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	var req ApplyPlanRequest
	_ = c.ShouldBindJSON(&req) // body is optional

	updates := planUserSubscriptionUpdates(p)
	updates["plan_id"] = p.ID
	if p.DurationDays > 0 {
		base := time.Now()
		if u.ExpiresAt.After(base) {
			base = u.ExpiresAt
		}
		updates["expires_at"] = base.AddDate(0, 0, p.DurationDays)
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&u).Updates(updates).Error; err != nil {
			return err
		}
		if req.Payment != nil {
			return CreatePayment(tx, u.ID, p.ID, p.DurationDays, c.GetString("admin"), *req.Payment)
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.First(&u, u.ID)
	auditAdminAction(c, "plan.apply_to_user", "user", u.ID, u.Username, map[string]interface{}{
		"plan_id":     p.ID,
		"plan_name":   p.Name,
		"has_payment": req.Payment != nil,
	})
	c.JSON(http.StatusOK, u)
}

type SetProxyPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

func SetProxyPassword(c *gin.Context) {
	var p model.Plan
	if database.DB.First(&p, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	var req SetProxyPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var syncedUsers int64
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&p).Update("proxy_password", req.Password).Error; err != nil {
			return err
		}
		p.ProxyPassword = req.Password
		var err error
		syncedUsers, err = syncPlanBoundUsers(tx, p)
		return err
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if p.ProxyHost != "" {
		service.ProbeOnce(p.ID)
	}
	auditAdminAction(c, "plan.set_proxy_password", "plan", p.ID, p.Name, map[string]interface{}{
		"synced_users": syncedUsers,
	})
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
