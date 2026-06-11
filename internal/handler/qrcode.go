package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	qrcode "github.com/skip2/go-qrcode"
)

func absSubURL(c *gin.Context, token string, format string) string {
	scheme := "https"
	if c.Request.TLS == nil && c.GetHeader("X-Forwarded-Proto") == "" {
		scheme = "http"
	}
	host := c.Request.Host
	if h := c.GetHeader("X-Forwarded-Host"); h != "" {
		host = h
	}
	url := scheme + "://" + host + "/api/sub/" + token
	if format != "" && format != "uri" {
		url += "?format=" + format
	}
	return url
}

func GetSubscriptionQRCode(c *gin.Context) {
	var u model.User
	if database.DB.First(&u, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	format := c.DefaultQuery("format", "uri")
	png, err := qrcode.Encode(absSubURL(c, u.SubToken, format), qrcode.Medium, 512)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "image/png", png)
}

func GetSubscriptionURLs(c *gin.Context) {
	var u model.User
	if database.DB.First(&u, c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"uri":               absSubURL(c, u.SubToken, "uri"),
		"clash":             absSubURL(c, u.SubToken, "clash"),
		"surge":             absSubURL(c, u.SubToken, "surge"),
		"shadowrocket":      absSubURL(c, u.SubToken, "shadowrocket"),
		"shadowrocket_conf": absSubURL(c, u.SubToken, "shadowrocket-conf"),
		"v2ray":             absSubURL(c, u.SubToken, "v2ray"),
	})
}
