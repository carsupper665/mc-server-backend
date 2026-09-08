package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go-backend/model"
	"go-backend/service"
)

func ModpackConfig(c *gin.Context) {
	c.JSON(200, gin.H{"max_size": service.MRPackLimit(), "blacklist": service.MRPackBlacklist()})
}

// Raw ZIP body avoids a second server-side preview/upload protocol.
func ImportModpack(c *gin.Context) {
	_, _, owner, _ := getPayloadAndId(c)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MRPackLimit())
	file, err := os.CreateTemp("", "mrpack-*.mrpack")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
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
		c.JSON(status, gin.H{"error": err.Error(), "max_size": service.MRPackLimit()})
		return
	}
	pack, err := service.ParseMRPack(file, size, service.MRPackBlacklist())
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	session, err := service.StartModpackInstall(owner, pack, file)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	accepted = true
	c.JSON(200, gin.H{"ins_ses_id": session.SessionID, "server_id": session.ServerID})
}

func ownedInstallJob(id string, owner uint) (*service.InstallJobSnapshot, error) {
	if strings.HasPrefix(id, "mrpack-") {
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
