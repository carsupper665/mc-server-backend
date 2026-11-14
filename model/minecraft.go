//model/minecraft.go

package model

import (
	"time"

	"gorm.io/gorm"
)

type UserMinecraftServer struct {
	OwnerID     uint   `gorm:"primaryKey;not null" json:"owner_id"`
	DisplayName string `gorm:"size:100;not null" json:"display_name"`
	ServerID    string `gorm:"primaryKey;size:32;not null" json:"server_id"`
	Args        string `gorm:"size:1000;default:''" json:"args"`
	Xms         int64  `gorm:"default:1073741824" json:"xms"` //minMemBytes
	Xmx         int64  `gorm:"default:2147483648" json:"xmx"` //maxMemBytes

	// new!! 伺服器配置
	MCVersion     string `gorm:"size:20" json:"mc_version"`     // "1.20.1"
	ModLoader     string `gorm:"size:20" json:"mod_loader"`     // "Vanilla"/"fabric"
	LoaderVersion string `gorm:"size:50" json:"loader_version"` // "47.2.0"

	InstallSessionID string `gorm:"index;size:50" json:"ins_ses_id,omitempty"`
	InstallStatus    string `gorm:"size:20" json:"install_status,omitempty"`
	InstallDetails   string `gorm:"type:text" json:"-"`
	SystemPath       string `gorm:"size:255;not null" json:"-"`
	// 關聯
	InstalledMods []ServerMod `gorm:"foreignKey:ServerID;references:ServerID"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func AddServerToUser(userID uint, serverID, displayName, ver, modLoader, loaderVer string, systemPath string, xms, xmx int64) error {
	userServer := UserMinecraftServer{
		OwnerID:       userID,
		ServerID:      serverID,
		DisplayName:   displayName,
		Xms:           xms,
		Xmx:           xmx,
		MCVersion:     ver,
		ModLoader:     modLoader,
		LoaderVersion: loaderVer,
		SystemPath:    systemPath,
	}
	return DB.Create(&userServer).Error
}

func GetUserServers(userID uint) ([]UserMinecraftServer, error) {
	var servers []UserMinecraftServer
	err := DB.Where("owner_id = ?", userID).Find(&servers).Error
	if err != nil {
		return nil, err
	}
	return servers, nil
}

func IsOwner(userID uint, serverID string) error {
	var servers []UserMinecraftServer
	err := DB.
		Where("owner_id = ? AND server_id = ?", userID, serverID).
		First(&servers).Error
	return err
}

func RemoveServerByServerID(userID uint, serverID string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&UserMinecraftServer{}).
			Where("owner_id = ? AND server_id = ?", userID, serverID).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		if err := tx.Where("server_id = ?", serverID).Delete(&ServerMod{}).Error; err != nil {
			return err
		}
		return tx.Where("owner_id = ? AND server_id = ?", userID, serverID).Delete(&UserMinecraftServer{}).Error
	})
}

func GetServerByID(userID uint, serverID string) (*UserMinecraftServer, error) {
	var server UserMinecraftServer
	err := DB.Where("owner_id = ? AND server_id = ?", userID, serverID).First(&server).Error
	if err != nil {
		return nil, err
	}
	return &server, nil
}

func GetServerCount(ownerID uint) (int64, error) {
	var count int64
	err := DB.Model(&UserMinecraftServer{}).Where("owner_id = ?", ownerID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func EditServerMemory(id uint, serverId string, xmx int64, xms *int64) error {
	updates := map[string]interface{}{"xmx": xmx}
	if xms != nil {
		updates["xms"] = *xms
	}
	result := DB.Model(&UserMinecraftServer{}).
		Where("owner_id = ? AND server_id = ?", id, serverId).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func EditServerArgs(id uint, serverId string, args string) error {
	result := DB.Model(&UserMinecraftServer{}).
		Where("owner_id = ? AND server_id = ?", id, serverId).
		Update("args", args)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
