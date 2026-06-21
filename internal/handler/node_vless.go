package handler

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ludandaye/hy2board/internal/config"
	"github.com/ludandaye/hy2board/internal/database"
	"github.com/ludandaye/hy2board/internal/model"
	"github.com/ludandaye/hy2board/internal/util"
)

// VlessClients returns the active VLESS users a node should accept.
// Auth: the Authorization header must equal the configured node secret.
// A user dropping out of this list (disabled/expired/over-limit) is how the
// on-node agent enforces revocation — same effect as HY2's connect-time IsActive().
func VlessClients(c *gin.Context) {
	if !config.C.HasNodeSecret() ||
		subtle.ConstantTimeCompare([]byte(c.GetHeader("Authorization")), []byte(config.C.Node.Secret)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var users []model.User
	database.DB.Find(&users)
	out := make([]gin.H, 0, len(users))
	for _, u := range users {
		if !u.IsActive() {
			continue
		}
		out = append(out, gin.H{
			"uuid":     util.VlessUUID(u.Username),
			"password": util.TrojanPassword(u.Username),
			"email":    u.Username,
		})
	}
	c.JSON(http.StatusOK, out)
}
