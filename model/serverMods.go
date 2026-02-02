package model

import (
	"errors"
	"go-backend/common"
	"time"

	"gorm.io/gorm"
)

// Mod 模組基本資訊表(從 Modrinth 快取)
type Mod struct {
	ModID       string `gorm:"primaryKey;size:50" json:"mod_id"` // Modrinth project_id
	Slug        string `gorm:"uniqueIndex;size:100" json:"slug"`
	Name        string `gorm:"size:200;not null" json:"name"`
	Summary     string `gorm:"type:text" json:"summary"`
	Description string `gorm:"type:text" json:"description"`

	// 作者資訊
	Author   string `gorm:"size:100" json:"author"`
	AuthorID string `gorm:"size:50" json:"author_id"`

	// 圖片
	IconURL   string `gorm:"size:500" json:"icon_url"`
	BannerURL string `gorm:"size:500" json:"banner_url"`

	// 統計
	Downloads int64 `gorm:"default:0" json:"downloads"`

	// 分類標籤
	Categories string `gorm:"type:json" json:"categories"` // ["technology", "utility"]
	Tags       string `gorm:"type:json" json:"tags"`       // ["performance", "optimization"]

	// Modrinth 原始資料(完整快取)
	ModrinthData string `gorm:"type:json" json:"modrinth_data,omitempty"`

	// 同步狀態
	ProjectUpdatedAt time.Time `json:"project_updated_at"`
	LastSynced       time.Time `json:"last_synced"`
	SyncStatus       string    `gorm:"size:20;default:'success'" json:"sync_status"` // success/failed

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 關聯
	Versions   []ModVersion `gorm:"foreignKey:ModID" json:"versions,omitempty"`
	ServerMods []ServerMod  `gorm:"foreignKey:ModID" json:"-"`
}

// ModVersion 模組版本表
type ModVersion struct {
	VersionID     string `gorm:"primaryKey;size:50" json:"version_id"` // Modrinth version_id
	ModID         string `gorm:"index;size:50;not null" json:"mod_id"`
	VersionNumber string `gorm:"size:50;not null" json:"version_number"` // "1.2.3"
	VersionName   string `gorm:"size:200" json:"version_name"`           // "Release 1.2.3"
	VersionType   string `gorm:"size:20" json:"version_type"`            // release/beta/alpha
	Changelog     string `gorm:"type:text" json:"changelog"`

	// 相容性資訊
	GameVersions string `gorm:"type:json;not null" json:"game_versions"` // ["1.20.1", "1.20.2"]

	// 下載資訊
	Files       string `gorm:"type:json" json:"files"`       // 檔案列表(完整資訊)
	PrimaryFile string `gorm:"size:500" json:"primary_file"` // 主要檔案名稱
	DownloadURL string `gorm:"size:1000" json:"download_url"`
	FileSize    int64  `json:"file_size"`
	FileHash    string `gorm:"size:64" json:"file_hash"` // SHA256

	// 依賴關係
	Dependencies string `gorm:"type:json" json:"dependencies"` // 依賴的其他模組

	//// 狀態
	Featured  bool  `gorm:"default:false" json:"featured"`
	Downloads int64 `gorm:"default:0" json:"downloads"`

	Published        time.Time `json:"published"`
	VersionUpdatedAt time.Time `json:"version_updated_at"`
	CreatedAt        time.Time `json:"created_at"`

	// 關聯
	Mod        Mod         `gorm:"foreignKey:ModID" json:"mod,omitempty"`
	ServerMods []ServerMod `gorm:"foreignKey:VersionID" json:"-"`
}

type ServerMod struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ServerID  string `gorm:"index:idx_server_mod,unique,priority:1;size:32;not null" json:"server_id"`
	ModID     string `gorm:"index:idx_server_mod,unique,priority:2;size:50;not null" json:"mod_id"`
	VersionID string `gorm:"size:50;not null" json:"version_id"` // 已安裝的版本
	Filename  string `gorm:"size:200" json:"file_name"`          // 檔案名稱

	// 安裝狀態
	Status       string `gorm:"size:20;default:'installed'" json:"status"` // installed/pending/failed/updating/disabled
	InstallPath  string `gorm:"size:500" json:"install_path"`              // 檔案實際路徑
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`

	// 更新資訊
	HasUpdate       bool      `gorm:"default:false" json:"has_update"`
	LatestVersionID string    `gorm:"size:50" json:"latest_version_id,omitempty"`
	UpdateCheckedAt time.Time `json:"update_checked_at,omitempty"`

	// 配置
	Enabled    bool `gorm:"default:true" json:"enabled"`
	AutoUpdate bool `gorm:"default:false" json:"auto_update"`

	// 時間戳
	InstalledAt time.Time `json:"installed_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 關聯
	Server        UserMinecraftServer `gorm:"foreignKey:ServerID;references:ServerID" json:"server,omitempty"`
	Mod           Mod                 `gorm:"foreignKey:ModID;references:ModID" json:"mod,omitempty"`
	Version       ModVersion          `gorm:"foreignKey:VersionID;references:VersionID" json:"version,omitempty"`
	LatestVersion *ModVersion         `gorm:"foreignKey:LatestVersionID;references:VersionID" json:"latest_version,omitempty"`
}

func GetServerMod(serverID, modID string) (*ServerMod, error) {
	var serverMod ServerMod
	err := DB.Where("server_id = ? AND mod_id = ?", serverID, modID).
		Preload("Server").
		Preload("LatestVersion").
		First(&serverMod).Error

	if err != nil {
		return nil, err
	}

	return &serverMod, nil
}

func GetModByID(modID string) (*Mod, error) {
	var mod Mod
	if err := DB.Where("mod_id = ?", modID).First(&mod).Error; err != nil {
		return nil, err
	}
	return &mod, nil
}

func GetModVersionByID(versionID string) (*ModVersion, error) {
	var version ModVersion
	if err := DB.Where("version_id = ?", versionID).First(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func UpsertMod(mod *Mod) (bool, error) {
	if mod == nil {
		return false, errors.New("mod is nil")
	}

	updated := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		var existing Mod
		err := tx.Where("mod_id = ?", mod.ModID).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				updated = true
				return tx.Create(mod).Error
			}
			return err
		}

		if shouldUpdateTime(existing.ProjectUpdatedAt, mod.ProjectUpdatedAt) {
			updated = true
			return tx.Model(&Mod{}).
				Where("mod_id = ?", mod.ModID).
				Updates(map[string]any{
					"slug":               mod.Slug,
					"name":               mod.Name,
					"summary":            mod.Summary,
					"description":        mod.Description,
					"author":             mod.Author,
					"author_id":          mod.AuthorID,
					"icon_url":           mod.IconURL,
					"banner_url":         mod.BannerURL,
					"downloads":          mod.Downloads,
					"categories":         mod.Categories,
					"tags":               mod.Tags,
					"modrinth_data":      mod.ModrinthData,
					"project_updated_at": mod.ProjectUpdatedAt,
					"last_synced":        mod.LastSynced,
					"sync_status":        mod.SyncStatus,
				}).Error
		}

		return nil
	})

	return updated, err
}

func UpsertModVersion(version *ModVersion) (bool, error) {
	if version == nil {
		return false, errors.New("mod version is nil")
	}

	updated := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		var existing ModVersion
		err := tx.Where("version_id = ?", version.VersionID).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				updated = true
				return tx.Create(version).Error
			}
			return err
		}

		if shouldUpdateTime(existing.VersionUpdatedAt, version.VersionUpdatedAt) {
			updated = true
			return tx.Model(&ModVersion{}).
				Where("version_id = ?", version.VersionID).
				Updates(map[string]any{
					"mod_id":             version.ModID,
					"version_number":     version.VersionNumber,
					"version_name":       version.VersionName,
					"version_type":       version.VersionType,
					"changelog":          version.Changelog,
					"game_versions":      version.GameVersions,
					"files":              version.Files,
					"primary_file":       version.PrimaryFile,
					"download_url":       version.DownloadURL,
					"file_size":          version.FileSize,
					"file_hash":          version.FileHash,
					"dependencies":       version.Dependencies,
					"featured":           version.Featured,
					"downloads":          version.Downloads,
					"published":          version.Published,
					"version_updated_at": version.VersionUpdatedAt,
				}).Error
		}

		return nil
	})

	return updated, err
}

func shouldUpdateTime(existing, incoming time.Time) bool {
	if incoming.IsZero() {
		return false
	}
	if existing.IsZero() {
		return true
	}
	return incoming.After(existing)
}

func ListMods(serverID string) ([]ServerMod, error) {
	var serverMods []ServerMod

	err := DB.
		Where("server_id = ?", serverID).
		Preload("Mod").Preload("Version").
		Find(&serverMods).
		Error
	if err != nil {
		return nil, err
	}

	return serverMods, nil
}

func DeleteMod(serverID, modID string) error {
	return DB.Where("server_id = ? AND mod_id = ?", serverID, modID).
		Delete(&ServerMod{}).Error
}

func AddModToServer(sid, modID, versionID, filename string, autoUpdate bool) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		exists, err := ModExists(sid, modID)
		if err != nil {
			return err
		}
		now := time.Now()
		if !exists {
			return tx.Create(&ServerMod{
				ServerID:    sid,
				ModID:       modID,
				VersionID:   versionID,
				Filename:    filename,
				Status:      "installed",
				Enabled:     true,
				AutoUpdate:  autoUpdate,
				InstalledAt: now,
				UpdatedAt:   now,
			}).Error
		}
		common.SysLog("Mod already exists for server, updating info, Server ID:" + sid + " Mod ID:" + modID)

		return tx.Model(&ServerMod{}).
			Where("server_id = ? AND mod_id = ?", sid, modID).
			Updates(map[string]any{
				"version_id":    versionID,
				"status":        "installed",
				"auto_update":   autoUpdate,
				"filename":      filename,
				"updated_at":    now,
				"error_message": "",
			}).Error
	})
}

func ModExists(sid, modID string) (bool, error) {
	var n int64
	err := DB.Model(&ServerMod{}).
		Where("server_id = ? AND mod_id = ?", sid, modID).
		Count(&n).Error
	return n > 0, err
}

func UpdateModState(sid, modID, filename, status string, enabled bool) error {
	return DB.Model(&ServerMod{}).
		Where("server_id = ? AND mod_id = ?", sid, modID).
		Updates(map[string]any{
			"filename":      filename,
			"status":        status,
			"enabled":       enabled,
			"error_message": "",
		}).Error
}

func ModIsEnable(sid, modID string) (bool, error) {
	var sm ServerMod

	err := DB.
		Model(&ServerMod{}).
		Select("enabled"). // 只撈需要的欄位
		Where("server_id = ? AND mod_id = ?", sid, modID).
		Take(&sm). // 找不到會回 gorm.ErrRecordNotFound
		Error

	if err != nil {
		return false, err
	}
	return sm.Enabled, nil
}
func ModFileName(sid, modID string) string {
	var sm ServerMod

	err := DB.
		Where("server_id = ? AND mod_id = ?", sid, modID).
		First(&sm).Error

	if err != nil {
		logger.Errorf("Failed to get server mod filename: %v", err)
		return ""
	}

	return sm.Filename
}
