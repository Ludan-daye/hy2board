package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
)

type RoutingRuleRequest struct {
	Enabled   *bool  `json:"enabled"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Value     string `json:"value"`
	Policy    string `json:"policy"`
	SortOrder int    `json:"sort_order"`
	Note      string `json:"note"`
}

func ListRoutingRules(c *gin.Context) {
	var rows []model.CustomRoutingRule
	database.DB.Order("sort_order asc, id asc").Find(&rows)
	c.JSON(http.StatusOK, rows)
}

func CreateRoutingRule(c *gin.Context) {
	rule, ok := bindRoutingRule(c)
	if !ok {
		return
	}
	if err := database.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	auditAdminAction(c, "routing_rule.create", "routing_rule", rule.ID, rule.Name, map[string]interface{}{
		"kind":   rule.Kind,
		"value":  rule.Value,
		"policy": rule.Policy,
	})
	c.JSON(http.StatusCreated, rule)
}

func UpdateRoutingRule(c *gin.Context) {
	var existing model.CustomRoutingRule
	if database.DB.First(&existing, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "routing rule not found"})
		return
	}
	rule, ok := bindRoutingRule(c)
	if !ok {
		return
	}
	updates := map[string]interface{}{
		"enabled":    rule.Enabled,
		"name":       rule.Name,
		"kind":       rule.Kind,
		"value":      rule.Value,
		"policy":     rule.Policy,
		"sort_order": rule.SortOrder,
		"note":       rule.Note,
	}
	if err := database.DB.Model(&existing).Updates(updates).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.First(&existing, existing.ID)
	auditAdminAction(c, "routing_rule.update", "routing_rule", existing.ID, existing.Name, map[string]interface{}{
		"fields": auditKeys(updates),
	})
	c.JSON(http.StatusOK, existing)
}

func DeleteRoutingRule(c *gin.Context) {
	var rule model.CustomRoutingRule
	if database.DB.First(&rule, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "routing rule not found"})
		return
	}
	database.DB.Delete(&rule)
	auditAdminAction(c, "routing_rule.delete", "routing_rule", rule.ID, rule.Name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func PreviewRoutingRule(c *gin.Context) {
	rule, ok := bindRoutingRule(c)
	if !ok {
		return
	}
	routingRule := service.CustomRoutingRuleToRoutingRule(rule)
	var nodes []model.Node
	database.DB.Where("healthy = ?", true).Order("sort_order asc, id asc").Find(&nodes)
	_, hasChain := service.EffectiveProxyChain(model.User{})
	hasHongKong := len(serviceHongKongNodeNames(nodes)) > 0
	line := service.ResolveRoutingRuleLine(routingRule, hasChain, hasHongKong)
	c.JSON(http.StatusOK, gin.H{
		"clash":        "  - " + line,
		"surge":        line,
		"shadowrocket": line,
	})
}

func bindRoutingRule(c *gin.Context) (model.CustomRoutingRule, bool) {
	var req RoutingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return model.CustomRoutingRule{}, false
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule := model.CustomRoutingRule{
		Enabled:   enabled,
		Name:      req.Name,
		Kind:      req.Kind,
		Value:     req.Value,
		Policy:    req.Policy,
		SortOrder: req.SortOrder,
		Note:      req.Note,
	}
	if err := service.ValidateCustomRoutingRule(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return model.CustomRoutingRule{}, false
	}
	return rule, true
}

func serviceHongKongNodeNames(nodes []model.Node) []string {
	names := make([]string, 0)
	for _, n := range nodes {
		if service.IsHongKongNodeName(n.Name) {
			names = append(names, n.Name)
		}
	}
	return names
}
