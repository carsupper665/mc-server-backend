// service/mcServerManager.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go-backend/common"
	"go-backend/model"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	Fabric  = "fabric"
	Vanilla = "vanilla"
)

type CreateServerRequest struct {
	ServerType      string `json:"server_type"`
	ServerVer       string `json:"server_ver"`
	FabricLoader    string `json:"fabric_loader"`
	FabricInstaller string `json:"fabric_installer"`
	DisplayName     string `json:"display_name"`
}

type GameVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

type ServerService struct {
	mgr *ServerManager
}

func ErrorFileClear(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("cleanup failed for %s: %w", path, err)
	}
	return nil
}

func NewServerService(mgr *ServerManager) *ServerService {
	return &ServerService{mgr: mgr}
}

func (s *ServerService) Start(sid, oid, workDir, maxMem, minMem string, args []string) (*Server, error) {
	return s.mgr.StartServer(sid, oid, workDir, maxMem, minMem, args)
}

func (s *ServerService) GetServerUsage(sid string) (Snapshot, error) {
	return s.mgr.GetServerUsage(sid)
}

func (s *ServerService) Stop(sid string) error {
	return s.mgr.StopServer(sid)
}

func (s *ServerService) Status(sid string) (string, error) {
	return s.mgr.GetServerStatus(sid)
}

func (s *ServerService) OwnerCount(oid string) int {
	return s.mgr.countByOwner(oid)
}

func (s *ServerService) ReadLatestLog(sid string) (string, error) {
	return s.mgr.ReadLatestLog(sid)
}

func (s *ServerService) SendCommand(sid string, command string) error {
	return s.mgr.SendCommand(sid, command)
}

func (s *ServerService) Backup(sid, workDir string) error {
	return s.mgr.BackUp(sid, workDir)
}

func (s *ServerService) RollBackSave(sid, file, workDir string) error {
	return s.mgr.ServerSaveRollBack(sid, file, workDir)
}

func (s *ServerService) DelServer(sid, workDir string) error {
	return s.mgr.DeleteServer(sid, workDir)
}

func (s *ServerService) ListBackups(workDir string) ([]string, error) {
	backups, err := s.mgr.ServerSaveList(workDir)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	return backups, err
}

func (s *ServerService) DeleteMod(sid, modID, modPath string) error {
	dbBk, err := model.GetServerMod(sid, modID)
	if err != nil {
		return err
	}
	if err := model.DeleteMod(sid, modID); err != nil {
		return err
	}
	if err := s.mgr.DeleteMod(sid, modPath); err != nil {
		_ = model.AddModToServer(
			sid,
			dbBk.ModID,
			dbBk.VersionID,
			dbBk.Filename,
			dbBk.AutoUpdate,
		)
		return err
	}
	return nil
}

func (s *ServerService) DisableMod(sid, modID, modPath string) error {
	mod, err := model.GetServerMod(sid, modID)
	if err != nil {
		return err
	}
	if err := s.mgr.DisableMod(sid, modPath); err != nil {
		return err
	}

	newFilename := mod.Filename + ".disable"
	if err := model.UpdateModState(sid, modID, newFilename, "disabled", false); err != nil {
		_ = s.mgr.EnableMod(sid, modPath+".disable")
		return err
	}

	return nil
}

func (s *ServerService) EnableMod(sid, modID, modPath string) error {
	mod, err := model.GetServerMod(sid, modID)
	if err != nil {
		return err
	}
	if err := s.mgr.EnableMod(sid, modPath); err != nil {
		return err
	}

	filename := strings.TrimSuffix(mod.Filename, ".disable")
	if err := model.UpdateModState(sid, modID, filename, "installed", true); err != nil {
		enabledPath := strings.TrimSuffix(modPath, filepath.Ext(modPath))
		_ = s.mgr.DisableMod(sid, enabledPath)
		return err
	}

	return nil
}

func (s *ServerService) UpdateMod(sid, modID, workDir, modPath, modLoader, gameVersion string, autoUpdate bool) error {
	//dbBk, err := model.GetServerMod(sid, modID)
	//if err != nil {
	//	return err
	//}
	err := s.mgr.UpdateMod(sid, modID, workDir, modPath, modLoader, gameVersion, autoUpdate)
	if err != nil {
		return err
	}
	return nil
}

func CreateServer(oid, serverType, serverVer, loader, installer string) (string, error) {
	serverID, workDir, err := prepareServerDirectory(serverType, oid, serverVer)
	if err != nil {
		return "", err
	}
	if err = installServer(context.Background(), workDir, serverType, serverVer, loader, installer); err != nil {
		_ = ErrorFileClear(workDir)
		return "", err
	}
	return serverID, nil
}

// prepareServerDirectory 建立伺服器資料夾並產生 Server ID。
//
// Server ID 格式固定為：<前綴><MC 版本>-<4 位隨機數字>-OID-<擁有者 uid>
// 例如 mcsvv-1.21.4-0427-OID-3、mcsfv-1.21.4-9135-OID-12。
// 前綴：Vanilla 為 "mcsvv-"，Fabric 為 "mcsfv-"。serverVer 一律傳 MC 版本（不是 loader 版本）。
//
// !!! 指定格式 別改!!!
func prepareServerDirectory(serverType, oid, serverVer string) (string, string, error) {
	prefix := "mcsvv-"
	if strings.ToLower(serverType) == Fabric {
		prefix = "mcsfv-"
	}
	if err := os.MkdirAll(common.MinecraftServerPath, 0755); err != nil {
		return "", "", err
	}
	// 4 位數字可能撞到同一使用者、同版本的既有伺服器，撞到就重抽，避免共用資料夾。
	for i := 0; i < 20; i++ {
		serverID := fmt.Sprintf("%s%s-%s-OID-%s", prefix, serverVer, common.GetRandomIntString(4), oid)
		workDir := filepath.Join(common.MinecraftServerPath, serverID)
		err := os.Mkdir(workDir, 0755)
		if err == nil {
			return serverID, workDir, nil
		}
		if !os.IsExist(err) {
			return "", "", err
		}
	}
	return "", "", errors.New("failed to allocate a unique server ID")
}

func installServer(ctx context.Context, workDir, serverType, serverVer, loader, installer string) error {
	if loader == "" {
		loader = common.LatestFabricLoaderVersion
	}
	_, uri, err := serverUri(serverType, serverVer, loader)
	if err != nil {
		return err
	}
	if strings.EqualFold(serverType, Fabric) && installer != "" {
		uri = fmt.Sprintf("https://meta.fabricmc.net/v2/versions/loader/%s/%s/%s/server/jar", serverVer, loader, installer)
	}
	if err := common.DownloadFileContext(ctx, http.DefaultClient, filepath.Join(workDir, "server.jar"), uri, nil, -1); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(workDir, "eula.txt"), []byte("eula=true\n"), 0644)
}

func serverUri(ServerType, serverVer, fabricLoader string) (string, string, error) {
	var err error
	var perFix, uri string
	var ok bool
	var fabricInstaller = common.LatestFabricInstallerVersion
	ServerType = strings.ToLower(ServerType)
	switch ServerType {
	case Fabric:
		perFix = "mcsfv-"
		uri = fmt.Sprintf(
			"https://meta.fabricmc.net/v2/versions/loader/%s/%s/%s/server/jar",
			serverVer, fabricLoader, fabricInstaller,
		)
	case Vanilla:
		perFix = "mcsvv-"
		uri, ok = common.VanillaServerUrl[serverVer]
		if !ok {
			err = errors.New("vanilla server url is empty")
			return "", "", err
		}
	default:
		err = errors.New("unsupported server type")
		return "", "", err
	}
	return perFix, uri, nil
}

func GetAllFabricVersions() ([]string, error) {
	resp, err := http.Get("https://meta.fabricmc.net/v2/versions/game")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get fabric versions, status code: %d", resp.StatusCode)
	}

	var gv []GameVersion
	if err := json.NewDecoder(resp.Body).Decode(&gv); err != nil {
		return nil, err
	}

	versions := make([]string, len(gv))
	for i, v := range gv {
		versions[i] = v.Version
	}
	return versions, nil
}

func GetAllVanillaVersions() (map[string]string, error) {
	all := common.VanillaServerUrl
	return all, nil
}
