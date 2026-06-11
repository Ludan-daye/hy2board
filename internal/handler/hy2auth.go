package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

type Hy2AuthRequest struct {
	Addr string `json:"addr"`
	Auth string `json:"auth"`
	TX   int64  `json:"tx"`
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

	c.JSON(http.StatusOK, gin.H{"ok": true, "id": user.Username})
}
