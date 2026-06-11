package handler

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
)

type auditLogResponse struct {
	Items []model.AuditLog `json:"items"`
	Total int64            `json:"total"`
}

func ListAuditLogs(c *gin.Context) {
	limit := parseAuditInt(c.Query("limit"), 80)
	if limit < 1 {
		limit = 80
	}
	if limit > 300 {
		limit = 300
	}
	offset := parseAuditInt(c.Query("offset"), 0)
	if offset < 0 {
		offset = 0
	}

	q := database.DB.Model(&model.AuditLog{})
	if v := c.Query("actor"); v != "" {
		q = q.Where("actor LIKE ?", "%"+v+"%")
	}
	if v := c.Query("action"); v != "" {
		q = q.Where("action = ?", v)
	}
	if v := c.Query("entity"); v != "" {
		q = q.Where("entity = ?", v)
	}

	var total int64
	q.Count(&total)

	var rows []model.AuditLog
	q.Order("id desc").Limit(limit).Offset(offset).Find(&rows)
	c.JSON(http.StatusOK, auditLogResponse{Items: rows, Total: total})
}

func parseAuditInt(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

func auditAdminAction(c *gin.Context, action, entity string, entityID uint, entityName string, detail map[string]interface{}) {
	_ = service.RecordAudit(database.DB, model.AuditLog{
		Actor:      c.GetString("admin"),
		Action:     action,
		Entity:     entity,
		EntityID:   entityID,
		EntityName: entityName,
		Detail:     service.AuditDetail(detail),
		IP:         c.ClientIP(),
	})
}

func auditKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
