package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-backend/common"
	"io"
	"net/http"
	"os"

	"go-backend/model"
	"go-backend/service"

	"github.com/gin-gonic/gin"
)

func ModpackConfig(c *gin.Context) {
	c.JSON(200, gin.H{"max_size": service.MRPackLimit(), "blacklist": service.MRPackBlacklist()})
}

// Raw ZIP body avoids a second server-side preview/upload protocol.
func ImportModpack(c *gin.Context) {
	var reqId any
	reqId = common.ReqId(c)
	_, _, owner, err := getPayloadAndId(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	if status, body, blocked := createServerLimitResponse(owner); blocked {
		c.JSON(status, body)
		return
	}
	xms, xmx, status, body := defaultMemoryForUser(owner)
	if status != 0 {
		c.JSON(status, body)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MRPackLimit())
	file, err := os.CreateTemp("", "mrpack-*.mrpack")
	if err != nil {
		logger.Errorf("create temp file: %v, req_id: %v", err, reqId)
		resMsg := fmt.Sprintf("Fail to Import Modpack, your Request Id %v", reqId)
		c.JSON(500, gin.H{"error": resMsg, "request_id": reqId})
		return
	}
	accepted := false
	defer func() {
		if !accepted {
			file.Close()
			os.Remove(file.Name())
		}
	}()
	size, err := io.Copy(file, c.Request.Body)
	if err != nil {
		status := 400
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			status = 413
		}
		resMsg := fmt.Sprintf("Fail to Import Modpack, check modpack size, your Request Id %v", reqId)
		c.JSON(status, gin.H{"error": resMsg, "max_size": service.MRPackLimit(), "request_id": reqId})
		logger.Errorf("Io copy error: %v, req_id: %v", err, reqId)
		return
	}
	pack, err := service.ParseMRPack(file, size, service.MRPackBlacklist())
	if err != nil {
		logger.Errorf("Mod pack parse error: %v, req_id: %v", err, reqId)
		c.JSON(400, gin.H{"error": "Mod Pack Parse Error", "request_id": reqId})
		return
	}
	if status, body, blocked := modpackModLimitResponse(owner, pack.ModJarCount()); blocked {
		c.JSON(status, body)
		return
	}
	// StartModpackInstall persists the server before returning. Serialize its final
	// check and insert with normal creation reservations; do not reserve it twice.
	serverCreationMu.Lock()
	if status, body, blocked := createServerLimitResponseLocked(owner); blocked {
		serverCreationMu.Unlock()
		c.JSON(status, body)
		return
	}
	session, err := service.StartModpackInstall(owner, pack, file, xms, xmx)
	serverCreationMu.Unlock()
	if err != nil {
		logger.Errorf("Fail to Start Mod Pack Install Error: %v, req_id: %v", err, reqId)
		if status, body, ok := versionErrorResponse(err); ok {
			body["request_id"] = reqId
			c.JSON(status, body)
			return
		}
		c.JSON(400, gin.H{"error": "Start Install Error", "request_id": reqId})
		return
	}
	accepted = true
	c.JSON(200, gin.H{"ins_ses_id": session.SessionID, "server_id": session.ServerID})
}

func ownedInstallJob(id string, owner uint) (*service.InstallJobSnapshot, error) {
	if service.IsServerInstallSession(id) {
		raw, err := service.GetModpackSession(id, owner)
		if err != nil {
			return nil, err
		}
		var job service.InstallJobSnapshot
		err = json.Unmarshal(raw, &job)
		return &job, err
	}
	job, ok := service.GetInstallJobSnapshot(id)
	if !ok {
		return nil, errors.New("job not found")
	}
	return job, model.IsOwner(owner, job.ServerID)
}
