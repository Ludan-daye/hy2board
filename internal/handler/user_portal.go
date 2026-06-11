package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/service"
	"github.com/ludandaye/hy2board/internal/util"
)

type UserLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func UserLogin(c *gin.Context) {
	var req UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var user model.User
	if database.DB.Where("username = ?", req.Username).First(&user).Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong credentials"})
		return
	}

	if user.LoginPassword == "" || !user.CheckLoginPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong credentials"})
		return
	}

	token, err := util.GenerateUserToken(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func UserMe(c *gin.Context) {
	username := c.GetString("user")

	var user model.User
	if database.DB.Where("username = ?", username).First(&user).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// All computed in background; this is a pure cache read.
	snaps := service.GetTrafficSnapshot()
	var totalTX, totalRX int64
	nodeTraffic := make([]gin.H, 0)

	for _, n := range snaps {
		if !n.Healthy || n.Traffic == nil {
			continue
		}
		for uid, t := range n.Traffic {
			if uid == user.Username || uid == user.Username+"_jp" || uid == user.Username+"_my" {
				totalTX += t.TX
				totalRX += t.RX
				nodeTraffic = append(nodeTraffic, gin.H{
					"node": n.Name,
					"tx":   t.TX,
					"rx":   t.RX,
				})
			}
		}
	}

	// Pre-computed per-user speed (bytes/sec)
	var speedTX, speedRX int64
	if s, ok := service.GetUserStat(user.Username); ok {
		speedTX = s.SpeedTX
		speedRX = s.SpeedRX
	}

	c.JSON(http.StatusOK, gin.H{
		"username":      user.Username,
		"sub_token":     user.SubToken,
		"traffic_limit": user.TrafficLimit,
		"traffic_used":  user.TrafficUsed,
		"traffic_tx":    totalTX,
		"traffic_rx":    totalRX,
		"speed_tx":      speedTX,
		"speed_rx":      speedRX,
		"expires_at":    user.ExpiresAt,
		"enabled":       user.Enabled,
		"active":        user.IsActive(),
		"nodes":         nodeTraffic,
		"history":       service.GetHistory(),
	})
}

func UserSetPassword(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
		return
	}

	var user model.User
	if database.DB.First(&user, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := user.SetLoginPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	database.DB.Model(&user).Update("login_password", user.LoginPassword)
	c.JSON(http.StatusOK, gin.H{"message": "password set"})
}
