//controller/minecraft.go

package controller

import (
	"errors"
	"fmt"
	"go-backend/common"
	"go-backend/model"
	"go-backend/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

func CreateServer(c *gin.Context) {
	var req service.CreateServerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		common.LogDebug(c.Request.Context(), "CreateMinecraftServer request binding error: "+err.Error())
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	_, uid_str, uid_uint, err := getPayloadAndId(c)

	serverID, err := service.CreateServer(uid_str, req.ServerType, req.ServerVer, req.FabricLoader, req.FabricInstaller)
	if err != nil {
		common.LogError(c.Request.Context(), "CreateMinecraftServer error: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to create server"})
		return
	}

	modelErr := model.AddServerToUser(uid_uint, serverID, req.DisplayName, common.MinecraftServerPath+"/"+serverID)
	if modelErr != nil {
		common.LogError(c.Request.Context(), "AddServerToUser error: "+modelErr.Error())
		service.ErrorFileClear(common.MinecraftServerPath + "/" + serverID)
		c.JSON(500, gin.H{"error": "Failed to add server to user"})
		return
	}

	c.JSON(200, gin.H{"server_id": serverID})
}

func GetAllVanillaVersions(c *gin.Context) {
	versions, err := service.GetAllVanillaVersions()
	if len(versions) == 0 || err != nil {
		c.JSON(404, gin.H{"error": ""})
		return
	}
	c.JSON(200, gin.H{"versions": versions})
}

func GetAllFabricVersions(c *gin.Context) {
	versions, err := service.GetAllFabricVersions()
	if len(versions) == 0 || err != nil {
		common.LogError(c.Request.Context(), "GetAllFabricVersions error: "+err.Error())
		c.JSON(404, gin.H{"error": ""})
		return
	}
	c.JSON(200, gin.H{"versions": versions})
}

func MyServers(c *gin.Context) {
	_, _, uid, err := getPayloadAndId(c)

	servers, err := model.GetUserServers(uid)
	if err != nil {
		common.LogDebug(c.Request.Context(), "GetUserServers error: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to retrieve servers"})
		return
	}

	c.JSON(200, servers)
}

func DeleteServerById(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}

	_, _, id_uint, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(500, gin.H{"error": "Internal Server Error: " + err.Error()})
		return
	}

	err = model.RemoveServerByServerID(id_uint, serverID)
	if err != nil {
		common.LogDebug(c.Request.Context(), "RemoveServerByServerID error: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to delete server"})
		return
	}

	c.JSON(200, gin.H{"message": "Server deleted successfully"})
}

func getPayloadAndId(c *gin.Context) (map[string]interface{}, string, uint, error) {
	token, err := c.Cookie(common.JwtCookieName)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to get JWT cookie: %w", err)
	}
	payload, err := common.GetJWTPayload(token)
	if err != nil {
		common.LogDebug(c.Request.Context(), "JWT error: "+err.Error())
		clearCookies(c)
		return nil, "", 0, fmt.Errorf("invalid token: %w", err)
	}

	rawUID, _ := payload["user_id"]
	uid, parseErr := strconv.ParseUint(rawUID.(string), 10, 32)

	if parseErr != nil {
		common.LogDebug(c.Request.Context(), "Parsing user id error: "+parseErr.Error())
		return nil, "", 0, fmt.Errorf("failed to parse user ID: %w", parseErr)
	}
	return payload, rawUID.(string), uint(uid), nil
}

// --------------------Server Controller--------------------

type ServerController struct {
	svc *service.ServerService
}

type SaveRollBackRequest struct {
	FileName string `json:"file_name" binding:"required"`
	ServerID string `json:"server_id" binding:"required"`
}

func (sc *ServerController) ListServerBackup(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}

	_, _, uintID, err := getPayloadAndId(c)

	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	serverInfo, err := model.GetServerByID(uintID, serverID)
	if err != nil {
		common.LogDebug(c.Request.Context(), "Log, GetServerByID error: "+err.Error())
		c.JSON(500, gin.H{"error": "Server start Failed."})
		return
	}

	backups, err := sc.svc.ListBackups(serverInfo.ServerID, serverInfo.SystemPath)
	if err != nil {
		common.LogDebug(c.Request.Context(), "ListBackups error: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to list backups"})
		return
	}

	c.JSON(200, gin.H{"backups": backups})
}

type Usages struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage uint64  `json:"memory_usage"`
	Threads     int32   `json:"threads"`
}

func (sc *ServerController) ServerUsage(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}

	_, _, uintId, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	if err := model.IsOwner(uintId, serverID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return

	}

	usage, err := sc.svc.GetServerUsage(serverID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(404, gin.H{"error": "Server not found"})
			return
		}
		common.LogDebug(c.Request.Context(), "GetServerUsage error: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to get server usage"})
		return
	}

	var usageResponse Usages
	usageResponse.CPUUsage = usage.CPU
	usageResponse.MemoryUsage = usage.VMS
	usageResponse.Threads = usage.Threads

	c.JSON(200, gin.H{"usage": usage})
}

func (sc *ServerController) SaveRollBack(c *gin.Context) {
	var req SaveRollBackRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		common.LogDebug(c.Request.Context(), "request binding error: "+err.Error())
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	_, _, uintID, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	if err := model.IsOwner(uintID, req.ServerID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
	}

	serverInfo, err := model.GetServerByID(uintID, req.ServerID)
	if err != nil {
		common.LogDebug(c.Request.Context(), "Log, GetServerByID error: "+err.Error())
		c.JSON(500, gin.H{"error": "Server start Failed."})
		return
	}

	err = sc.svc.RollBackSave(req.ServerID, req.FileName, serverInfo.SystemPath)

	if err != nil {
		if !errors.Is(err, service.ErrServerRunning) {
			common.LogError(c.Request.Context(), "RollBackSave error: "+err.Error())
			c.JSON(500, gin.H{"error": "Failed to save server to user"})
			return
		}
		c.JSON(500, gin.H{"error": "Cannot rollback while server is running"})
		return
	}

	c.Status(200)
}

func NewServerController(svc *service.ServerService) *ServerController {
	return &ServerController{svc: svc}
}

func (sc *ServerController) GetServerLog(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}

	_, _, uintID, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	serverInfo, err := model.GetServerByID(uintID, serverID)
	if err != nil {
		common.LogDebug(c.Request.Context(), "Log, GetServerByID error: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to retrieve server log"})
		return
	}

	logs, err := sc.svc.ReadLatestLog(serverInfo.ServerID)
	if err != nil {
		common.LogDebug(c.Request.Context(), "Log, GetServerLog error: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to retrieve server log"})
		return
	}
	c.JSON(200, gin.H{"logs": logs})
}

func (sc *ServerController) GetStatus(c *gin.Context) {
	// Get the server status
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}
	status, err := sc.svc.Status(serverID)
	if err != nil {
		common.LogDebug(c.Request.Context(), "Log, GetStatus error: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to get server status, or server not found."})
		return
	}
	c.JSON(200, gin.H{"status": status})

}

func (sc *ServerController) Start(c *gin.Context) {

	sid := c.Param("server_id")
	if sid == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}

	_, oid, uintID, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	serverInfo, err := model.GetServerByID(uintID, sid)
	if err != nil {
		common.LogDebug(c.Request.Context(), "Log, GetServerByID error: "+err.Error())
		c.JSON(500, gin.H{"error": "Server start Failed."})
		return
	}

	srv, err := sc.svc.Start(sid, oid, serverInfo.SystemPath, "2G", "1G", []string{})
	if err != nil {
		common.LogDebug(c.Request.Context(), "Log, StartServer error: "+err.Error())
		if !errors.Is(err, service.ErrAlreadyRunning) && !errors.Is(err, service.ErrNotFound) && !errors.Is(err, service.ErrMaxReached) {
			common.LogError(c.Request.Context(), "Log, StartServer error: "+err.Error())
		}
		c.JSON(500, gin.H{"error": "Failed to start server"})
		return
	}
	c.JSON(200, gin.H{"message": "Server started successfully", "server_id": srv.ID})
}

func (sc *ServerController) Stop(c *gin.Context) {
	sid := c.Param("server_id")
	if sid == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}

	_, _, uintID, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	serverInfo, err := model.GetServerByID(uintID, sid)
	if err != nil {
		common.LogDebug(c.Request.Context(), "Log, GetServerByID error: "+err.Error())
		c.JSON(500, gin.H{"error": "Server Stop Failed."})
		return
	}

	err = sc.svc.Stop(serverInfo.ServerID)
	if err != nil {
		common.LogDebug(c.Request.Context(), "Log, StopServer error: "+err.Error())
		if !errors.Is(err, service.ErrAlreadyRunning) && !errors.Is(err, service.ErrNotFound) && !errors.Is(err, service.ErrMaxReached) {
			common.LogError(c.Request.Context(), "Log, StopServer error: "+err.Error())
		}
		c.JSON(500, gin.H{"error": "Failed Stop Server"})
		return
	}

	c.JSON(200, gin.H{"message": "Server is stopped"})
}

func (sc *ServerController) GetServerProperties(c *gin.Context) {
	sid := c.Param("server_id")
	if sid == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}

	_, _, uintID, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	serverInfo, err := model.GetServerByID(uintID, sid)
	if err != nil {
		common.LogDebug(c.Request.Context(), "Log, GetServerByID error: "+err.Error())
		c.JSON(500, gin.H{"error": "Server Stop Failed."})
		return
	}

	texts, err := service.GetPropertyText(serverInfo.SystemPath)
	if err != nil {
		common.LogDebug(c.Request.Context(), "Get Property fail. err: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to get server properties."})
		return
	}
	c.JSON(200, gin.H{"message": "Property Get.", "property": texts})
}

type SendCommandRequest struct {
	Command string `json:"command" binding:"required"`
}

func (sc *ServerController) SendCommand(c *gin.Context) {
	var req SendCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.LogDebug(c.Request.Context(), "request binding error: "+err.Error())
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	sid := c.Param("server_id")
	if sid == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}

	_, _, uintID, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	serverInfo, err := model.GetServerByID(uintID, sid)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get server information."})
		return
	}

	err = sc.svc.SendCommand(serverInfo.ServerID, req.Command)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to send command to server."})
		return
	}

	c.JSON(200, gin.H{"message": "Command sent successfully."})
}

func (sc *ServerController) Backup(c *gin.Context) {
	sid := c.Param("server_id")
	if sid == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}

	_, _, uintID, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	serverInfo, err := model.GetServerByID(uintID, sid)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get server information."})
		return
	}

	err = sc.svc.Backup(serverInfo.ServerID, serverInfo.SystemPath)

	if err != nil {
		if !errors.Is(err, service.ErrServerRunning) {
			common.LogError(c.Request.Context(), "Backup error: "+err.Error())
		}
		r_id := c.Request.Context().Value(common.RequestIdKey)
		c.JSON(500, gin.H{"error": "Failed to backup server. Request id: " + r_id.(string)})
		return
	}

	c.Status(200)

}

type UploadPropertyRequest struct {
	Texts string `json:"texts" binding:"required"`
}

func (sc *ServerController) UploadProperty(c *gin.Context) {
	var req UploadPropertyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		common.LogDebug(c.Request.Context(), "request binding error: "+err.Error())
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	sid := c.Param("server_id")
	if sid == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}

	_, _, uintID, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	serverInfo, err := model.GetServerByID(uintID, sid)
	if err != nil {
		c.JSON(500, gin.H{"error": "Upload Error: " + err.Error()})
		return
	}

	err = service.ReplaceProperty(serverInfo.SystemPath, req.Texts)
	if err != nil {
		c.JSON(500, gin.H{"error": "Upload Error: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Uploaded."})

}
