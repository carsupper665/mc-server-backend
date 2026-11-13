package controller

import (
	"github.com/gin-gonic/gin"
	"go-backend/common"
	"go-backend/model"
	"net/http"
)

func CurrentUser(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	userID := c.GetUint("uintId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	perm, _ := common.PermFor(user.Role)
	unlimited := common.IsRoot(user.Role)
	c.JSON(http.StatusOK, gin.H{
		"id": user.ID, "username": user.Username, "display_name": user.DisplayName, "role": user.Role,
		"permissions": gin.H{
			"unlimited":            unlimited,
			"server_max_mem":       perm.ServerMaxMem,
			"max_server_count":     perm.MaxServerCount,
			"max_online_count":     perm.MaxOnlineCount,
			"max_mods_count":       perm.MaxModsCount,
			"can_add_user":         unlimited || perm.CanAddUser,
			"can_create_server":    unlimited || perm.CanCreateServer,
			"can_edit_server_args": unlimited || perm.CanEditServerArgs,
			"can_back_up_server":   unlimited || perm.CanBackUpServer,
			"max_back_up_count":    perm.MaxBackUpCount,
		},
	})
}
