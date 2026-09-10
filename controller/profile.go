package controller

import (
	"github.com/gin-gonic/gin"
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
	c.JSON(http.StatusOK, gin.H{"id": user.ID, "username": user.Username, "display_name": user.DisplayName, "role": user.Role})
}
