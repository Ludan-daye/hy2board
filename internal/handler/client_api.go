package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
)

func AppVersion(c *gin.Context) {
	c.JSON(http.StatusOK, service.BuildClientVersion(time.Now().UTC()))
}

func AppFeatures(c *gin.Context) {
	c.JSON(http.StatusOK, service.BuildClientFeatures())
}

func currentClientUser(c *gin.Context) (model.User, bool) {
	username := c.GetString("user")
	var user model.User
	if username == "" || database.DB.Where("username = ?", username).First(&user).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return model.User{}, false
	}
	return user, true
}

func clientPlanForUser(u model.User) *model.Plan {
	if u.PlanID == nil {
		return nil
	}
	var plan model.Plan
	if database.DB.First(&plan, *u.PlanID).Error != nil {
		return nil
	}
	return &plan
}

func clientProbeStates() map[uint]model.NodeProbeState {
	var states []model.NodeProbeState
	database.DB.Find(&states)
	out := make(map[uint]model.NodeProbeState, len(states))
	for _, state := range states {
		out[state.NodeID] = state
	}
	return out
}

func clientVisibleNodes(u model.User, healthyOnly bool) []model.Node {
	var nodes []model.Node
	q := database.DB.Order("sort_order asc, id asc")
	if healthyOnly {
		q = q.Where("healthy = ?", true)
	}
	q.Find(&nodes)
	if allowed := u.EffectiveNodeIDs(); allowed != nil {
		filtered := make([]model.Node, 0, len(nodes))
		for _, n := range nodes {
			if allowed[n.ID] {
				filtered = append(filtered, n)
			}
		}
		nodes = filtered
	}
	return nodes
}

func ClientSession(c *gin.Context) {
	user, ok := currentClientUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"server_time":            time.Now().UTC(),
		"api_version":            service.ClientAPIVersion,
		"minimum_client_version": service.MinimumClientVersion,
		"authenticated":          true,
		"profile":                service.BuildClientProfile(user),
	})
}

func ClientBootstrap(c *gin.Context) {
	user, ok := currentClientUser(c)
	if !ok {
		return
	}
	now := time.Now().UTC()
	plan := clientPlanForUser(user)
	states := clientProbeStates()
	nodes := clientVisibleNodes(user, true)
	allVisible := clientVisibleNodes(user, false)
	unavailable := 0
	for _, n := range allVisible {
		if !n.Healthy {
			unavailable++
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"session": gin.H{
			"server_time":            now,
			"api_version":            service.ClientAPIVersion,
			"minimum_client_version": service.MinimumClientVersion,
			"authenticated":          true,
		},
		"profile":       service.BuildClientProfile(user),
		"plan":          service.BuildClientPlan(user, plan),
		"traffic":       service.BuildClientTrafficSummary(user),
		"nodes":         service.BuildClientNodes(nodes, states),
		"announcements": service.BuildClientAnnouncements(now),
		"help":          service.BuildClientHelp(),
		"features":      service.BuildClientFeatures(),
		"diagnostics":   service.BuildClientDiagnostics(user, len(nodes), unavailable),
	})
}

func ClientProfile(c *gin.Context) {
	user, ok := currentClientUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, service.BuildClientProfile(user))
}

func ClientPlan(c *gin.Context) {
	user, ok := currentClientUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, service.BuildClientPlan(user, clientPlanForUser(user)))
}

func ClientTrafficSummary(c *gin.Context) {
	user, ok := currentClientUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, service.BuildClientTrafficSummary(user))
}

func ClientTrafficHistory(c *gin.Context) {
	user, ok := currentClientUser(c)
	if !ok {
		return
	}
	days := 7
	if d, err := strconv.Atoi(c.DefaultQuery("days", "7")); err == nil && d > 0 && d <= 365 {
		days = d
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	var logs []model.TrafficLog
	database.DB.Where("username = ? AND sampled_at >= ?", user.Username, cutoff).
		Order("sampled_at asc, id asc").
		Find(&logs)
	c.JSON(http.StatusOK, service.BuildClientTrafficHistory(logs))
}

func ClientTrafficNodes(c *gin.Context) {
	user, ok := currentClientUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, service.BuildClientNodeTraffic(user.Username))
}

func ClientNodes(c *gin.Context) {
	user, ok := currentClientUser(c)
	if !ok {
		return
	}
	states := clientProbeStates()
	nodes := clientVisibleNodes(user, true)
	c.JSON(http.StatusOK, service.BuildClientNodes(nodes, states))
}

func ClientConfig(c *gin.Context) {
	user, ok := currentClientUser(c)
	if !ok {
		return
	}
	states := clientProbeStates()
	nodes := clientVisibleNodes(user, true)
	c.JSON(http.StatusOK, service.BuildClientConfig(user, nodes, states))
}

func ClientAnnouncements(c *gin.Context) {
	c.JSON(http.StatusOK, service.BuildClientAnnouncements(time.Now().UTC()))
}

func ClientHelp(c *gin.Context) {
	c.JSON(http.StatusOK, service.BuildClientHelp())
}

func ClientDiagnostics(c *gin.Context) {
	user, ok := currentClientUser(c)
	if !ok {
		return
	}
	nodes := clientVisibleNodes(user, true)
	allVisible := clientVisibleNodes(user, false)
	unavailable := 0
	for _, n := range allVisible {
		if !n.Healthy {
			unavailable++
		}
	}
	c.JSON(http.StatusOK, service.BuildClientDiagnostics(user, len(nodes), unavailable))
}
