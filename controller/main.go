package controller

import (
	"fmt"
	"go-backend/common"

	"github.com/gin-gonic/gin"
)

var logger *common.SysLogger

func InitLogger() error {
	logger = common.Logger
	if logger == nil {
		return fmt.Errorf("logger not initialized")
	}
	return nil
}

func reqid(c *gin.Context) string {
	ctx := c.Request.Context()
	return ctx.Value(common.RequestIdKey).(string)
}
