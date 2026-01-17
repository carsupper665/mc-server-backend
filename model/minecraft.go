//model/minecraft.go

package model

import (
	"time"
)

type UserMinecraftServer struct {
	OwnerID     uint   `gorm:"primaryKey;not null" json:"owner_id"`
	DisplayName string `gorm:"size:100;not null" json:"display_name"`
	ServerID    string `gorm:"primaryKey;size:32;not null" json:"server_id"`

	// new!! 伺服器配置
	MCVersion     string `gorm:"size:20" json:"mc_version"`     // "1.20.1"
	ModLoader     string `gorm:"size:20" json:"mod_loader"`     // "Vanilla"/"fabric"
	LoaderVersion string `gorm:"size:50" json:"loader_version"` // "47.2.0"

	SystemPath string `gorm:"size:255;not null" json:"system_path"`
	// 關聯
	InstalledMods []ServerMod `gorm:"foreignKey:ServerID;references:ServerID"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func AddServerToUser(userID uint, serverID, displayName, ver, modLoader, loaderVer string, systemPath string) error {
	userServer := UserMinecraftServer{
		OwnerID:       userID,
		ServerID:      serverID,
		DisplayName:   displayName,
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
	return DB.Where("owner_id = ? AND server_id = ?", userID, serverID).Delete(&UserMinecraftServer{}).Error
}

func GetServerByID(userID uint, serverID string) (*UserMinecraftServer, error) {
	var server UserMinecraftServer
	err := DB.Where("owner_id = ? AND server_id = ?", userID, serverID).First(&server).Error
	if err != nil {
		return nil, err
	}
	return &server, nil
}
