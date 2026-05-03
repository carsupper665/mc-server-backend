package controller

import (
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
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     int    `json:"role" binding:"required"`
}

func NewUser(c *gin.Context) {
	if err := c.ShouldBindJSON(&RegisterUserRequest{}); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
}
