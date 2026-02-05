package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-backend/common"
	"go-backend/model"
	"go-backend/service"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	sysPath := common.MinecraftServerPath + "/" + serverID

	modelErr := model.AddServerToUser(uid_uint, serverID, req.DisplayName, req.ServerVer,
		req.ServerType, req.FabricLoader, sysPath)

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

func ServerDetails(c *gin.Context) {
	sid := c.Param("server_id")
	if sid == "" {
		c.JSON(404, gin.H{"error": "Server Not Found"})
		return
	}
	_, _, uid, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": ""})
		return
	}
	if err := model.IsOwner(uid, sid); err != nil {
		c.JSON(403, gin.H{"error": "Forbidden"})
		return
	}

	server, err := model.GetServerByID(uid, sid)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to retrieve server"})
		return
	}
	c.JSON(200, gin.H{"server": server})
}

func getPayloadAndId(c *gin.Context) (map[string]any, string, uint, error) {
	payload := c.MustGet("payload").(map[string]any)
	rawUID := c.MustGet("stringId").(string)
	uid := c.MustGet("uintId").(uint)

	return payload, rawUID, uid, nil
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

	backups, err := sc.svc.ListBackups(serverInfo.SystemPath)
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

func parseMemBytes(raw string) (int64, error) {
	s := strings.TrimSpace(strings.ToUpper(raw))
	if s == "" {
		return 0, errors.New("empty memory size")
	}

	unit := int64(1)
	switch {
	case strings.HasSuffix(s, "KB"):
		unit = 1024
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "K"):
		unit = 1024
		s = strings.TrimSuffix(s, "K")
	case strings.HasSuffix(s, "MB"):
		unit = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "M"):
		unit = 1024 * 1024
		s = strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "GB"):
		unit = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "G"):
		unit = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "G")
	case strings.HasSuffix(s, "TB"):
		unit = 1024 * 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "TB")
	case strings.HasSuffix(s, "T"):
		unit = 1024 * 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "T")
	}

	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("invalid memory size")
	}
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil || val < 0 {
		return 0, errors.New("invalid memory size")
	}
	return val * unit, nil
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
	// 兼容舊DB 沒有設定 XMX XMS
	xmxBytes := serverInfo.Xmx
	xmsBytes := serverInfo.Xms
	if xmxBytes < 0 {
		c.JSON(400, gin.H{"error": "Invalid Xmx value"})
		return
	}
	if xmsBytes < 0 {
		c.JSON(400, gin.H{"error": "Invalid Xms value"})
		return
	}
	if xmxBytes == 0 {
		xmxBytes = 2 * 1024 * 1024 * 1024
	}
	if xmsBytes == 0 {
		xmsBytes = 1024 * 1024 * 1024
	}
	if xmsBytes > xmxBytes {
		c.JSON(400, gin.H{"error": "Xms cannot be greater than Xmx"})
		return
	}

	xmxArg := fmt.Sprintf("%dM", xmxBytes/common.MB)
	xmsArg := fmt.Sprintf("%dM", xmsBytes/common.MB)
	srv, err := sc.svc.Start(sid, oid, serverInfo.SystemPath, xmxArg, xmsArg, []string{})
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
		c.JSON(500, gin.H{"error": "Upload Error"})
		return
	}

	err = service.ReplaceProperty(serverInfo.SystemPath, req.Texts)
	if err != nil {
		c.JSON(500, gin.H{"error": "Upload Error"})
		return
	}

	c.JSON(200, gin.H{"message": "Uploaded."})

}

func (sc *ServerController) DeleteServerById(c *gin.Context) {
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

	err = model.IsOwner(id_uint, serverID)
	if err != nil {
		c.JSON(500, gin.H{"error": "You are not the owner of this server"})
		return
	}

	serverData, err := model.GetServerByID(id_uint, serverID)
	if err != nil {
		common.LogDebug(c.Request.Context(), "GetServerByID error: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to retrieve server information"})
		return
	}

	err = sc.svc.DelServer(serverID, serverData.SystemPath)
	if err != nil {
		common.LogDebug(c.Request.Context(), "DelServer error: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to delete server files"})
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

// Mods Manager

type AddModRequest struct {
	ModID      string `json:"mod_id" binding:"required"`
	VersionID  string `json:"version_id"`
	AutoUpdate bool   `json:"auto_update"` // 是否自動更新
}

func (sc *ServerController) AddMod(c *gin.Context) {
	var req AddModRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	sid := c.Param("server_id")
	if sid == "" {
		c.JSON(400, gin.H{"error": "Server ID is required"})
		return
	}
	_, _, userId, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	if err := model.IsOwner(userId, sid); err != nil {
		c.JSON(400, gin.H{"error": "Not the owner of the server"})
		return
	}

	serverData, err := model.GetServerByID(userId, sid)
	if err != nil {
		logger.Debugf("Add mod error, GetServerByID: %v", err)
		c.JSON(500, gin.H{"error": "Failed to retrieve server information"})
		return
	}

	if c.Query("async") == "true" {
		job, err := service.EnqueueModInstall(
			sid,
			serverData.SystemPath,
			serverData.ModLoader,
			serverData.MCVersion,
			req.ModID,
			req.VersionID,
			req.AutoUpdate,
		)
		if err != nil {
			common.LogError(c.Request.Context(), "install Mod error: "+err.Error())
			c.JSON(500, gin.H{"error": "Failed to add mod."})
			return
		}
		c.JSON(200, gin.H{"job_id": job.ID, "status": job.Status})
		return
	}

	if err := service.EnqueueModInstallAndWait(
		sid,
		serverData.SystemPath,
		serverData.ModLoader,
		serverData.MCVersion,
		req.ModID,
		req.VersionID,
		req.AutoUpdate,
	); err != nil {
		common.LogError(c.Request.Context(), "install Mod error: "+err.Error())
		c.JSON(500, gin.H{"error": "Failed to add mod."})
		return
	}

	c.JSON(200, gin.H{"message": "mod installed Successfully"})
}

func (sc *ServerController) GetModInstallJob(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(400, gin.H{"error": "job id is required"})
		return
	}
	_, _, userId, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	job, ok := service.GetInstallJobSnapshot(jobID)
	if !ok {
		c.JSON(404, gin.H{"error": "job not found"})
		return
	}
	if err := model.IsOwner(userId, job.ServerID); err != nil {
		c.JSON(403, gin.H{"error": "Not the owner of the server"})
		return
	}

	c.JSON(200, job)
}

func (sc *ServerController) SubscribeModInstall(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(400, gin.H{"error": "job id is required"})
		return
	}
	_, _, userId, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	job, ok := service.GetInstallJobSnapshot(jobID)
	if !ok {
		c.JSON(404, gin.H{"error": "job not found"})
		return
	}
	if err := model.IsOwner(userId, job.ServerID); err != nil {
		c.JSON(403, gin.H{"error": "Not the owner of the server"})
		return
	}

	events, history, unsubscribe, err := service.SubscribeInstallEvents(jobID)
	if err != nil {
		c.JSON(404, gin.H{"error": "job not found"})
		return
	}
	defer unsubscribe()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(500, gin.H{"error": "streaming unsupported"})
		return
	}

	sendEvent := func(ev service.InstallEvent) error {
		payload, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(c.Writer, "id: %d\n", ev.ID)
		_, _ = fmt.Fprintf(c.Writer, "event: progress\n")
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		flusher.Flush()
		return nil
	}

	for _, ev := range history {
		_ = sendEvent(ev)
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			_ = sendEvent(ev)
		case <-heartbeat.C:
			_, _ = fmt.Fprintf(c.Writer, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func (sc *ServerController) ToggleMod(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(400, gin.H{"error": "Invalid server ID"})
		return
	}
	modID := c.Query("mod_id")
	if modID == "" {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	_, _, userId, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	err = model.IsOwner(userId, serverID)
	if err != nil {
		c.JSON(400, gin.H{"error": "Not the owner of the server"})
		return
	}

	serverData, err := model.GetServerByID(userId, serverID)
	if err != nil {
		logger.Errorf("Disable mod error, GetServerByID: %v, ReqId: %s", err, reqid(c))
		c.JSON(500, gin.H{"error": "Failed to retrieve server information"})
		return
	}

	rid := reqid(c) // for log

	// DB get mod file
	modFile := model.ModFileName(serverID, modID)
	if modFile == "" {
		c.JSON(500, gin.H{"error": "Mod File Not Found or error"})
		return
	}
	modPath := filepath.Join(serverData.SystemPath, "mods", modFile)
	var f func(sid, mid, mp string) error

	if ok, err := model.ModIsEnable(serverID, modID); err != nil {
		logger.Errorf("Disable mod error, \nat model\a at ModIsEnable \n Error: %v, ReqId: %s", err, rid)
		c.JSON(500, gin.H{"error": "Failed to toggle mod."})
		return
	} else if !ok {
		f = sc.svc.EnableMod
	} else {
		f = sc.svc.DisableMod
	}

	if err := f(serverID, modID, modPath); err != nil {
		logger.Errorf("Disable mod error: %v, ReqId: %s", err, rid)
		c.JSON(500, gin.H{"error": "Failed to disable mod."})
		return
	}

	c.JSON(200, gin.H{"message": "mod toggle successfully"})
}

func (sc *ServerController) ListMod(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(400, gin.H{"error": "Invalid server ID"})
		return
	}

	_, _, userId, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	if err := model.IsOwner(userId, serverID); err != nil {
		logger.Errorf("List mod error, \n at model IsOwner %v", err)
		c.JSON(400, gin.H{"error": "Not the owner of the server, or server does not exist"})
		return
	}
	var modData []model.ServerMod
	modData, err = model.ListMods(serverID)
	if err != nil {
		logger.Errorf("List mod error, \n at model ListMods %v", err)
		c.JSON(500, gin.H{"error": "Failed to retrieve server information"})
		return
	}

	type modInfo struct {
		Name         string `json:"name"`
		ModID        string `json:"mod_id"`
		Version      string `json:"version"`
		Enabled      bool   `json:"enabled"`
		GameVersions string `json:"game_versions"`

		IconURL    string `json:"icon_url"`
		Summary    string `json:"summary"`
		Categories string `json:"categories"`
	}
	resp := make([]modInfo, 0, len(modData))
	for _, sm := range modData {
		name := sm.Mod.Name
		if name == "" {
			name = sm.ModID
		}
		version := sm.Version.VersionNumber
		if version == "" {
			version = sm.Version.VersionName
		}
		if version == "" {
			version = sm.VersionID
		}
		gameVersions := sm.Version.GameVersions

		resp = append(resp, modInfo{
			Name:         name,
			ModID:        sm.ModID,
			Version:      version,
			Enabled:      sm.Enabled,
			GameVersions: gameVersions,
			IconURL:      sm.Mod.IconURL,
			Summary:      sm.Mod.Summary,
			Categories:   sm.Mod.Categories,
		})
	}

	c.JSON(200, gin.H{"data": resp})
}

func (sc *ServerController) DelMod(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(400, gin.H{"error": "Invalid server ID"})
		return
	}

	modID := c.Query("mod_id")
	if modID == "" {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	_, _, userId, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	err = model.IsOwner(userId, serverID)
	if err != nil {
		c.JSON(403, gin.H{"error": "Not the owner of the server"})
		return
	}
	serverData, err := model.GetServerByID(userId, serverID)
	fileName := model.ModFileName(serverID, modID)
	modPath := filepath.Join(serverData.SystemPath, "mods", fileName)

	if err := sc.svc.DeleteMod(serverID, modID, modPath); err != nil {
		logger.Errorf("Delete mod error, \n	at service DeleteMod %v", err)
		c.JSON(500, gin.H{"error": "Failed to delete mod."})
		return
	}
	c.JSON(200, gin.H{"message": "mod deleted successfully"})
}

func (sc *ServerController) UpdateMod(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(400, gin.H{"error": "Invalid server ID"})
		return
	}

	modID := c.Query("mod_id")
	if modID == "" {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	_, _, userId, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	err = model.IsOwner(userId, serverID)
	if err != nil {
		c.JSON(403, gin.H{"error": "Not the owner of the server"})
		return
	}
	serverData, err := model.GetServerByID(userId, serverID)
	if err != nil {
		logger.Errorf("Update mod error, \n at model GetServerByID %v", err)
		c.JSON(500, gin.H{"error": "Failed to retrieve server information"})
		return
	}
	modInf, err := model.GetServerMod(serverID, modID)
	if err != nil {
		logger.Errorf("Update mod error, \n at model GetServerMod %v", err)
		c.JSON(500, gin.H{"error": "Failed to retrieve server information"})
		return
	}

	modPath := filepath.Join(serverData.SystemPath, "mods", modInf.Filename)
	err = sc.svc.UpdateMod(serverID, modID, serverData.SystemPath, modPath, serverData.ModLoader, serverData.MCVersion, modInf.AutoUpdate)
	if err != nil {
		logger.Errorf("Update mod error, \n at service UpdateMod %v", err)
		c.JSON(500, gin.H{"error": "Failed to update mod."})
		return
	}

	c.JSON(200, gin.H{"message": "mod updated successfully"})
}
