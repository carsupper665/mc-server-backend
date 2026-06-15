package controller

import (
	"go-backend/common"
	"go-backend/model"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminVerify struct {
	Code int
	Aud  string
	Exp  time.Time
}

type AdminVerifyStorage struct {
	Cache map[uint]AdminVerify // userId uint
	mu    sync.RWMutex
}

func reSetAdminToken(userId uint) error {
	// 生成新的 admin token
	return nil
}

type RegisterUserRequest struct {
	Username    string `json:"username" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	Role        int    `json:"role" binding:"required"`
}

type EditUserReq struct {
	UserId         uint   `json:"user_id" binding:"required"`
	NewPassword    string `json:"new_password" binding:"optional"`
	NewEmail       string `json:"new_email" binding:"optional"`
	NewUsername    string `json:"new_username" binding:"optional"`
	NewDisplayName string `json:"new_display_name" binding:"optional"`
	NewRole        int    `json:"new_role" binding:"optional"`
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
	if err := model.AddUser(req.Username, req.Email, req.DisplayName, req.Password, req.Role); err != nil {
		c.JSON(500, gin.H{"message": "Failed to create user", "request_id": c.Request.Context().Value(common.RequestIdKey)})
		common.Logger.Errorf("Failed to create user: %v", err)
		return
	}
	common.Logger.Infof("New User Add, UserName: %s, Role: %d, Password: %s", req.Username, req.Role, req.Password)
	c.JSON(200, gin.H{"message": "User created successfully"})
	return
}

func EditUser(c *gin.Context) {
	var req EditUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": "Invalid request"})
		return
	}
	err := model.UpdateUser(req.UserId, req.NewUsername, req.NewDisplayName, req.NewEmail, req.NewRole)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to update user", "request_id": c.Request.Context().Value(common.RequestIdKey)})
		common.Logger.Errorf("Failed to update user: %v", err)
		return
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
	c.JSON(200, gin.H{"data": user})
}
