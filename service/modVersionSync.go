package service

import (
	"fmt"
	"go-backend/common"
	"go-backend/model"
	"strings"
)

func SyncInstalledModVersions() error {
	targets, err := model.ListInstalledModSyncTargets()
	if err != nil {
		return fmt.Errorf("list installed mod sync targets: %w", err)
	}

	if len(targets) == 0 {
		if common.Logger != nil {
			common.Logger.Info("Mod version auto-sync skipped: no installed mods")
		}
		return nil
	}

	synced := 0
	failures := make([]string, 0)

	for _, target := range targets {
		version, err := getLatestOrSpecific(target.ModID, target.ModLoader, target.MCVersion, "", false) // 先預設為false 此功能待測試
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s(%s/%s): %v", target.ModID, target.ModLoader, target.MCVersion, err))
			continue
		}

		if err := upsertModVersion(target.ModID, version); err != nil {
			failures = append(failures, fmt.Sprintf("%s(%s/%s): %v", target.ModID, target.ModLoader, target.MCVersion, err))
			continue
		}

		synced++
	}

	if common.Logger != nil {
		common.Logger.Infof("Mod version auto-sync finished: targets=%d synced=%d failed=%d", len(targets), synced, len(failures))
	}

	if len(failures) > 0 {
		return fmt.Errorf("mod version auto-sync partial failure: %s", strings.Join(failures, "; "))
	}

	return nil
}
