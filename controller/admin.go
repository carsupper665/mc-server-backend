package controller

import (
	"go-backend/common"
	"go-backend/model"

	"github.com/gin-gonic/gin"
)

type RegisterUserRequest struct {
	Username    string `json:"username" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	Role        *int   `json:"role" binding:"required"`
}

type EditUserReq struct {
	UserId         uint   `json:"user_id" binding:"required"`
	NewPassword    string `json:"new_password"`
	NewEmail       string `json:"new_email"`
	NewUsername    string `json:"new_username"`
	NewDisplayName string `json:"new_display_name"`
	NewRole        *int   `json:"new_role"`
}

type ResetAccReq struct {
	UserId      uint   `json:"user_id" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func NewUser(c *gin.Context) {
	var req *RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request"})
		return
	}
	if !common.IsAssignableRole(*req.Role) {
		c.JSON(400, gin.H{"message": "Invalid role"})
		return
	}
	if err := model.AddUser(req.Username, req.Email, req.DisplayName, req.Password, *req.Role); err != nil {
		c.JSON(500, gin.H{"message": "Failed to create user", "request_id": common.ReqId(c)})
		common.Logger.Errorf("Failed to create user: %v", err)
		return
	}
	common.Logger.Infof("New User Add, UserName: %s, Role: %d", req.Username, *req.Role)
	c.JSON(200, gin.H{"message": "User created successfully"})
	return
}

func EditUser(c *gin.Context) {
	var req EditUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request"})
		return
	}
	role := -1
	if req.NewRole != nil {
		if !common.IsAssignableRole(*req.NewRole) {
			c.JSON(400, gin.H{"message": "Invalid role"})
			return
		}
		role = *req.NewRole
	}
	if req.NewPassword != "" && len(req.NewPassword) < 8 {
		c.JSON(400, gin.H{"message": "Invalid request"})
		return
	}
	if req.NewRole != nil || req.NewDisplayName != "" {
		var target model.User
		if err := model.DB.First(&target, req.UserId).Error; err == nil && target.FGFSubject != nil {
			c.JSON(400, gin.H{"error": "FGF 帳號的角色與顯示名稱由 FGF 同步，無法手動修改", "code": "ADMIN_FGF_MANAGED"})
			return
		}
	}
	err := model.UpdateUser(req.UserId, req.NewUsername, req.NewDisplayName, req.NewEmail, role)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to update user", "request_id": c.Request.Context().Value(common.RequestIdKey)})
		common.Logger.Errorf("Failed to update user: %v", err)
		return
	}
	if req.NewPassword != "" {
		if err := model.UpdatePassword(req.UserId, req.NewPassword); err != nil {
			c.JSON(500, gin.H{"message": "Failed to update user", "request_id": c.Request.Context().Value(common.RequestIdKey)})
			common.Logger.Errorf("Failed to update password: %v", err)
			return
		}
	}
	common.Logger.Infof("User ID: %d, Update Successful", req.UserId)
	c.JSON(200, gin.H{"message": "User updated successfully"})
	return
}

func ResetAccount(c *gin.Context) {
	var req *ResetAccReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request"})
		return
	}
	if err := model.UpdatePassword(req.UserId, req.NewPassword); err != nil {
		c.JSON(500, gin.H{"message": "Failed to reset password", "request_id": c.Request.Context().Value(common.RequestIdKey)})
		common.Logger.Errorf("Failed to reset password: %v", err)
		return
	}
	common.Logger.Infof("Reset Account Successful")
	c.JSON(200, gin.H{"message": "User reset successfully"})
	return
}

func GetAllUser(c *gin.Context) {
	user, err := model.GetAllUserData()
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to get user", "request_id": c.Request.Context().Value(common.RequestIdKey)})
		common.Logger.Errorf("Failed to get user: %v", err)
		return
	}
	type userItem struct {
		model.User
		IsFGF bool `json:"is_fgf"`
	}
	data := make([]userItem, len(user))
	for i, u := range user {
		data[i] = userItem{User: u, IsFGF: u.FGFSubject != nil}
	}
	c.JSON(200, gin.H{"data": data})
}
