package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
)

func GetUserTrafficHistory(c *gin.Context) {
	var u model.User
	if database.DB.First(&u, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	days := 7
	if d, err := strconv.Atoi(c.DefaultQuery("days", "7")); err == nil && d > 0 && d <= 365 {
		days = d
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	var logs []model.TrafficLog
	database.DB.Where("username = ? AND sampled_at >= ?", u.Username, cutoff).
		Order("sampled_at asc").Find(&logs)
	c.JSON(http.StatusOK, logs)
}
