// service/mcServerManager.go
package service

import (
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
	FabricLoader    string `json:"fabric_loader"`  // 舊欄位，等同 loader_version
	LoaderVersion   string `json:"loader_version"` // Fabric／Forge／NeoForge 版本；空白時用預設（Forge 推薦版、其他最新穩定版）
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

func (s *ServerService) Start(sid, oid, workDir string, launch Launch) (*Server, error) {
	return s.mgr.StartServer(sid, oid, workDir, launch)
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

// serverIDPrefixes 是各伺服器類型的 Server ID 前綴（使用者指定，只能新增，不要改既有的）。
var serverIDPrefixes = map[string]string{
	Vanilla:  "mcsvv-",
	Fabric:   "mcsfv-",
	Forge:    "mcsgv-",
	NeoForge: "mcsnv-",
}

// prepareServerDirectory 建立伺服器資料夾並產生 Server ID。
//
// Server ID 格式固定為：<前綴><MC 版本>-<4 位隨機數字>-OID-<擁有者 uid>
// 例如 mcsvv-1.21.4-0427-OID-3、mcsfv-1.21.4-9135-OID-12。
// 前綴：Vanilla 為 "mcsvv-"，Fabric 為 "mcsfv-"，Forge 為 "mcsgv-"，NeoForge 為 "mcsnv-"。
// serverVer 一律傳 MC 版本（不是 loader 版本）。
//
// !!! 指定格式 別改!!!
func prepareServerDirectory(serverType, oid, serverVer string) (string, string, error) {
	prefix, ok := serverIDPrefixes[strings.ToLower(serverType)]
	if !ok {
		prefix = serverIDPrefixes[Vanilla]
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
