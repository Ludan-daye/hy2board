package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/service"
)

func GetTelegramStatus(c *gin.Context) {
	c.JSON(http.StatusOK, service.GetTelegramStatus())
}

func SendTelegramAdminTestNotice(c *gin.Context) {
	if err := service.SendTestAdminNewMemberNotice(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditAdminAction(c, "telegram.test_admin_notice", "telegram", 0, "admin notice", nil)
	c.JSON(http.StatusOK, gin.H{"message": "admin test notice sent"})
}

func SendTelegramTestPost(c *gin.Context) {
	if err := service.SendTestDailyPost(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditAdminAction(c, "telegram.test_daily_post", "telegram", 0, "daily post", nil)
	c.JSON(http.StatusOK, gin.H{"message": "test post sent"})
}

func SendTelegramActivityAnnouncement(c *gin.Context) {
	if err := service.PostAndPinActivity(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditAdminAction(c, "telegram.announce_activity", "telegram", 0, "activity announcement", nil)
	c.JSON(http.StatusOK, gin.H{"message": "activity announcement sent and pinned"})
}
