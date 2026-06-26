package handler

import (
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
)

type Hy2AuthRequest struct {
	Addr string `json:"addr"`
	Auth string `json:"auth"`
	TX   int64  `json:"tx"`
}

// ipFromAddr strips the port from HY2's client address, leaving the bare host/IP.
func ipFromAddr(addr string) string {
	if addr == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

func Hy2Auth(c *gin.Context) {
	var req Hy2AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false})
		return
	}

	var user model.User
	if database.DB.Where("hy2_password = ?", req.Auth).First(&user).Error != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false})
		return
	}

	if !user.IsActive() {
		c.JSON(http.StatusOK, gin.H{"ok": false})
		return
	}

	// Per-user IP concurrency limit (0 = unlimited). Record the source IP either way so the
	// admin UI can show current IP counts; only block when a limit is set and exceeded.
	if ip := ipFromAddr(req.Addr); ip != "" {
		allowed, _ := service.TouchIP(user.ID, ip, user.MaxIPs, time.Now())
		if user.MaxIPs > 0 && !allowed {
			c.JSON(http.StatusOK, gin.H{"ok": false})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "id": user.Username})
}
