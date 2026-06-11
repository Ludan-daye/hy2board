package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
)

func Subscribe(c *gin.Context) {
	token := c.Param("token")

	var user model.User
	if database.DB.Where("sub_token = ?", token).First(&user).Error != nil {
		c.String(http.StatusNotFound, "invalid subscription")
		return
	}

	if !user.IsActive() {
		c.String(http.StatusForbidden, "subscription expired or disabled")
		return
	}

	var nodes []model.Node
	database.DB.Where("healthy = ?", true).Order("sort_order asc").Find(&nodes)

	// Apply per-user node whitelist (no-op if user.NodeIDs is "all" or empty).
	if allowed := user.EffectiveNodeIDs(); allowed != nil {
		filtered := make([]model.Node, 0, len(nodes))
		for _, n := range nodes {
			if allowed[n.ID] {
				filtered = append(filtered, n)
			}
		}
		nodes = filtered
	}

	if len(nodes) == 0 {
		c.String(http.StatusServiceUnavailable, "no available nodes")
		return
	}

	customRules, err := service.ListEnabledCustomRoutingRules(database.DB)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load routing rules")
		return
	}

	format := c.DefaultQuery("format", "")
	if format == "" {
		ua := strings.ToLower(c.GetHeader("User-Agent"))
		switch {
		case strings.Contains(ua, "shadowrocket"):
			format = "shadowrocket"
		case strings.Contains(ua, "v2ray"), strings.Contains(ua, "v2box"), strings.Contains(ua, "nekobox"):
			format = "v2ray"
		case strings.Contains(ua, "surge"):
			format = "surge"
		case strings.Contains(ua, "clash"), strings.Contains(ua, "mihomo"):
			format = "clash"
		default:
			format = "uri"
		}
	}

	var content string
	switch format {
	case "shadowrocket", "sr":
		content = service.GenerateShadowrocketNodes(user, nodes)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	case "shadowrocket-conf", "sr-conf":
		content = service.GenerateShadowrocketWithCustomRules(user, nodes, customRules)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	case "v2ray", "v2":
		raw := service.GenerateV2RayNURI(user, nodes)
		content = base64.StdEncoding.EncodeToString([]byte(raw))
		c.Header("Content-Type", "text/plain; charset=utf-8")
	case "surge":
		content = service.GenerateSurgeWithCustomRules(user, nodes, customRules)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	case "clash":
		content = service.GenerateClashWithCustomRules(user, nodes, customRules)
		c.Header("Content-Type", "text/yaml; charset=utf-8")
	default:
		content = service.GenerateURI(user, nodes)
		c.Header("Content-Type", "text/plain; charset=utf-8")
	}

	// Subscription-userinfo header — Clash/Surge/Shadowrocket convention.
	// Split the single TrafficUsed counter 50/50 until Phase 2 wires up per-direction live data.
	var expireUnix int64
	if !user.ExpiresAt.IsZero() {
		expireUnix = user.ExpiresAt.Unix()
	}
	upload := user.TrafficUsed / 2
	download := user.TrafficUsed - upload
	c.Header("subscription-userinfo",
		fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%d",
			upload, download, user.TrafficLimit, expireUnix))
	c.Header("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s.conf"`, user.Username))
	c.String(http.StatusOK, content)
}
