package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

type createCostRequest struct {
	Name        string     `json:"name" binding:"required"`
	Category    string     `json:"category"`
	AmountCents int64      `json:"amount_cents"`
	Note        string     `json:"note"`
	IncurredAt  *time.Time `json:"incurred_at,omitempty"`
}

type updateCostRequest struct {
	Name        *string    `json:"name,omitempty"`
	Category    *string    `json:"category,omitempty"`
	AmountCents *int64     `json:"amount_cents,omitempty"`
	Note        *string    `json:"note,omitempty"`
	IncurredAt  *time.Time `json:"incurred_at,omitempty"`
}

func ListCosts(c *gin.Context) {
	q := database.DB.Model(&model.Cost{})
	if v := c.Query("category"); v != "" {
		q = q.Where("category = ?", v)
	}
	if v := c.Query("from"); v != "" {
		q = q.Where("incurred_at >= ?", v)
	}
	if v := c.Query("to"); v != "" {
		q = q.Where("incurred_at < ?", v+" 23:59:59")
	}

	var total int64
	q.Count(&total)

	page := 1
	size := 50
	if v := c.Query("page"); v != "" {
		fmt.Sscanf(v, "%d", &page)
	}
	if v := c.Query("size"); v != "" {
		fmt.Sscanf(v, "%d", &size)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 500 {
		size = 50
	}

	var items []model.Cost
	q.Order("incurred_at DESC, id DESC").Limit(size).Offset((page - 1) * size).Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "size": size})
}

func CreateCost(c *gin.Context) {
	var req createCostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	incurredAt := time.Now().UTC()
	if req.IncurredAt != nil {
		incurredAt = req.IncurredAt.UTC()
	}
	cost := model.Cost{
		Name:        req.Name,
		Category:    req.Category,
		AmountCents: req.AmountCents,
		Note:        req.Note,
		Operator:    c.GetString("admin"),
		IncurredAt:  incurredAt,
	}
	if err := database.DB.Create(&cost).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	auditAdminAction(c, "cost.create", "cost", cost.ID, cost.Name, map[string]interface{}{
		"amount_cents": cost.AmountCents,
		"category":     cost.Category,
	})
	c.JSON(http.StatusCreated, cost)
}

func UpdateCost(c *gin.Context) {
	var cost model.Cost
	if database.DB.First(&cost, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cost not found"})
		return
	}
	var req updateCostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.AmountCents != nil {
		updates["amount_cents"] = *req.AmountCents
	}
	if req.Note != nil {
		updates["note"] = *req.Note
	}
	if req.IncurredAt != nil {
		updates["incurred_at"] = (*req.IncurredAt).UTC()
	}
	if len(updates) == 0 {
		c.JSON(http.StatusOK, cost)
		return
	}
	if err := database.DB.Model(&cost).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.First(&cost, cost.ID)
	auditAdminAction(c, "cost.update", "cost", cost.ID, cost.Name, map[string]interface{}{
		"fields": auditKeys(updates),
	})
	c.JSON(http.StatusOK, cost)
}

func DeleteCost(c *gin.Context) {
	var cost model.Cost
	if database.DB.First(&cost, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cost not found"})
		return
	}
	database.DB.Delete(&cost)
	auditAdminAction(c, "cost.delete", "cost", cost.ID, cost.Name, map[string]interface{}{
		"amount_cents": cost.AmountCents,
		"category":     cost.Category,
	})
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
