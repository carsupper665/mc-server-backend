package service

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"strings"
	"sync"
	"time"

	"go-backend/common"
	"go-backend/model"

	"github.com/google/uuid"
)

type ModpackSession struct {
	InstallJobSnapshot
	SessionID string         `json:"ins_ses_id"`
	OwnerID   uint           `json:"owner_id"`
	Details   []InstallEvent `json:"details"`
	mu        sync.Mutex
	cancel    context.CancelFunc
	done      chan struct{}
}

type modpackSessionStore struct {
	sync.Mutex
	items    map[string]*ModpackSession
	stopping bool
}

var modpackSessions = &modpackSessionStore{items: map[string]*ModpackSession{}}

// The archive file belongs to the session after this succeeds.
func StartModpackInstall(owner uint, pack *MRPack, archive *os.File, xms, xmx int64) (*ModpackSession, error) {
	loader, version, err := pack.Loader()
	if err != nil {
		return nil, err
	}
	if modpackSessions.isStopping() {
		return nil, errors.New("backend is shutting down")
	}
	resolveCtx, cancelResolve := context.WithTimeout(context.Background(), versionFetchTimeout)
	src, err := ResolveServerSource(resolveCtx, loader, pack.Dependencies["minecraft"], version, "")
	cancelResolve()
	if err != nil {
		return nil, err
	}
	return startInstallSession(installSessionSpec{
		owner: owner, prefix: modpackSessionPrefix, displayName: pack.Name,
		modLoader: loader, src: src, xms: xms, xmx: xmx,
		run: func(ctx context.Context, s *ModpackSession, workDir string) error {
			return s.install(ctx, pack, workDir, src)
		},
		after: func() {
			os.Remove(archive.Name())
			archive.Close()
		},
	})
}

type installSessionSpec struct {
	owner          uint
	prefix         string
	displayName    string
	modLoader      string
	src            *ServerSource
	xms, xmx       int64
	clearOnFailure bool
	run            func(ctx context.Context, s *ModpackSession, workDir string) error
	after          func()
}

func (m *modpackSessionStore) isStopping() bool {
	m.Lock()
	defer m.Unlock()
	return m.stopping
}

// startInstallSession persists an installing server and runs spec.run in the background.
// Modpack imports and server creation share it, including progress, history and SSE.
func startInstallSession(spec installSessionSpec) (*ModpackSession, error) {
	modpackSessions.Lock()
	defer modpackSessions.Unlock()
	if modpackSessions.stopping {
		return nil, errors.New("backend is shutting down")
	}
	owner, src := spec.owner, spec.src
	// 統一格式：Server ID 一律用 MC 版本
	serverID, workDir, err := prepareServerDirectory(src.Type, fmt.Sprintf("%d", owner), src.MCVersion)
	if err != nil {
		return nil, err
	}
	if err := model.AddServerToUser(owner, serverID, spec.displayName, src.MCVersion, spec.modLoader, src.LoaderVersion, workDir, spec.xms, spec.xmx); err != nil {
		_ = ErrorFileClear(workDir)
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	id := spec.prefix + uuid.NewString()
	session := &ModpackSession{
		InstallJobSnapshot: InstallJobSnapshot{ID: id, ServerID: serverID, Status: JobQueued, CreatedAt: time.Now()},
		SessionID:          id, OwnerID: owner, Details: []InstallEvent{}, cancel: cancel, done: make(chan struct{}),
	}
	session.record("queued", "", 0, "install session created", nil)
	if err := session.save(); err != nil {
		cancel()
		_ = model.RemoveServerByServerID(owner, serverID)
		_ = ErrorFileClear(workDir)
		CloseInstallEvents(id)
		modInstallEvents.mu.Lock()
		delete(modInstallEvents.streams, id)
		modInstallEvents.mu.Unlock()
		return nil, err
	}
	modpackSessions.items[id] = session
	go func() {
		defer close(session.done)
		defer cancel()
		if spec.after != nil {
			defer spec.after()
		}
		session.mu.Lock()
		session.StartedAt = time.Now()
		session.Status = JobRunning
		session.mu.Unlock()
		err := spec.run(ctx, session, workDir)
		status, message := "completed", "server installed"
		if spec.prefix == modpackSessionPrefix {
			message = "modpack installed"
		}
		if err != nil {
			status, message = "failed", err.Error()
		}
		if ctx.Err() != nil {
			status, message, err = "interrupted", "backend shutdown interrupted installation", ctx.Err()
		}
		if err != nil && spec.clearOnFailure {
			if clearErr := clearFailedInstall(workDir); clearErr != nil {
				versionWarn("clear failed install: " + clearErr.Error())
			}
		}
		session.record(status, "", session.percent(), message, err)
		if err := session.save(); err != nil {
			session.record("failed", "", session.percent(), "could not save installation history", err)
			common.SysError("save modpack session: " + err.Error())
		}
		session.mu.Lock()
		publishInstallEvent(session.Details[len(session.Details)-1])
		session.mu.Unlock()
		CloseInstallEvents(id)
	}()
	return session, nil
}

func (s *ModpackSession) percent() int { s.mu.Lock(); defer s.mu.Unlock(); return s.Progress }

func (s *ModpackSession) record(stage, file string, percent int, message string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event := InstallEvent{JobID: s.ID, ServerID: s.ServerID, Stage: stage, ModName: file, Percent: percent, Message: message, Timestamp: time.Now()}
	if err != nil {
		event.Error = err.Error()
	}
	s.Progress, s.Message = percent, message
	switch stage {
	case "completed", "failed", "interrupted":
		s.Status, s.EndedAt, s.Error = InstallJobStatus(stage), time.Now(), event.Error
		if stage == "completed" {
			s.Progress, event.Percent = 100, 100
		}
	}
	event.ID = int64(len(s.Details) + 1)
	s.Details = append(s.Details, event)
	if stage != "completed" && stage != "failed" && stage != "interrupted" {
		publishInstallEvent(event)
	}
}

func (s *ModpackSession) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	result := model.DB.Model(&model.UserMinecraftServer{}).Where("server_id = ? AND owner_id = ?", s.ServerID, s.OwnerID).
		Updates(map[string]any{"install_session_id": s.ID, "install_status": s.Status, "install_details": string(data)})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("session server record missing")
	}
	return nil
}

func GetModpackSession(id string, owner uint) (json.RawMessage, error) {
	modpackSessions.Lock()
	s := modpackSessions.items[id]
	modpackSessions.Unlock()
	if s != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.OwnerID != owner {
			return nil, errors.New("session not found")
		}
		return json.Marshal(s)
	}
	var server model.UserMinecraftServer
	if err := model.DB.Where("install_session_id = ? AND owner_id = ?", id, owner).First(&server).Error; err != nil {
		return nil, err
	}
	return json.RawMessage(server.InstallDetails), nil
}

func (s *ModpackSession) install(ctx context.Context, pack *MRPack, workDir string, src *ServerSource) error {
	for _, file := range pack.Files {
		s.record("queued", file.Path, 0, "file selected", nil)
	}
	for _, file := range pack.Overrides {
		s.record("queued", file.Name, 0, "override selected", nil)
	}
	s.record("creating", "", 0, "downloading "+src.Type+" server", nil)
	// Server progress stays at 0% here; the file downloads below drive the percentage.
	result, err := installServer(ctx, workDir, src, func(stage string, _ int, message string) { s.record("creating", "", 0, message, nil) })
	if err != nil {
		return err
	}
	if err := model.SetServerLaunch(s.OwnerID, s.ServerID, result.Mode, result.File); err != nil {
		return err
	}
	for _, name := range pack.Skipped {
		s.record("skipped", name, 0, "excluded by server environment or blacklist", nil)
	}
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		return packURL(req.URL.String())
	}}
	total := len(pack.Files) + len(pack.Overrides)
	done := 0
	for _, file := range pack.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		s.record("downloading", file.Path, done*100/total, file.Path, nil)
		var err error
		for _, url := range file.Downloads {
			err = common.DownloadFileContext(ctx, client, filepath.Join(workDir, filepath.FromSlash(file.Path)), url, file.Hashes, file.FileSize)
			if err == nil {
				break
			}
			s.record("retrying", file.Path, done*100/total, "download source failed: "+err.Error(), nil)
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
		if err != nil {
			return fmt.Errorf("%s: %w", file.Path, err)
		}
		if strings.HasPrefix(file.Path, "mods/") && strings.HasSuffix(file.Path, ".jar") {
			if err := s.cacheMod(ctx, file); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				s.record("metadata", file.Path, done*100/total, "file installed; Modrinth metadata unavailable: "+err.Error(), nil)
			}
		}
		done++
		s.record("installed", file.Path, done*100/total, file.Path, nil)
	}
	for _, entry := range pack.Overrides {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, name, _ := strings.Cut(entry.Name, "/")
		dest := filepath.Join(workDir, filepath.FromSlash(name))
		s.record("overriding", entry.Name, done*100/total, entry.Name, nil)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		r, err := entry.Open()
		if err != nil {
			return err
		}
		err = copyPackOverride(ctx, dest, r)
		r.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", entry.Name, err)
		}
		if strings.HasPrefix(name, "mods/") && strings.HasSuffix(name, ".jar") {
			file, err := os.Open(dest)
			if err != nil {
				return err
			}
			hash := sha512.New()
			_, err = io.Copy(hash, file)
			file.Close()
			if err != nil {
				return err
			}
			// An override replaces any earlier file and its corresponding mod record.
			if err := model.DB.Where("server_id = ? AND filename = ?", s.ServerID, strings.TrimPrefix(name, "mods/")).Delete(&model.ServerMod{}).Error; err != nil {
				return err
			}
			if err := s.cacheMod(ctx, MRPackFile{Path: name, Hashes: map[string]string{"sha512": fmt.Sprintf("%x", hash.Sum(nil))}}); err != nil {
				// Keep every installed jar in the quota even when no project is known.
				// A local identifier also lets existing mod endpoints address it safely.
				filename := strings.TrimPrefix(name, "mods/")
				if err := model.AddModToServer(s.ServerID, model.LocalModID(filename), "", filename, false); err != nil {
					return fmt.Errorf("record override mod %s: %w", filename, err)
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
				s.record("metadata", entry.Name, done*100/total, "file installed; Modrinth metadata unavailable: "+err.Error(), nil)
			}
		}
		done++
		s.record("installed", entry.Name, done*100/total, entry.Name, nil)
	}
	return ctx.Err()
}

func copyPackOverride(ctx context.Context, dest string, reader io.Reader) error {
	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()
	buffer := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := reader.Read(buffer)
		if n > 0 {
			if _, writeErr := file.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err == io.EOF {
			return file.Close()
		}
		if err != nil {
			return err
		}
	}
}

// Hash lookup enriches installed mods; it never changes the pack's selected file/version.
func (s *ModpackSession) cacheMod(ctx context.Context, file MRPackFile) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.modrinth.com/v2/version_file/"+file.Hashes["sha512"]+"?algorithm=sha512", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "mc-server-backend/mrpack")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("metadata HTTP %d", resp.StatusCode)
	}
	var version ModrinthVersion
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return err
	}
	if version.ProjectID == "" || version.ID == "" {
		return errors.New("missing project/version ID")
	}
	if _, err := model.UpsertMod(&model.Mod{ModID: version.ProjectID, Slug: version.ProjectID, Name: filepath.Base(file.Path), SyncStatus: "partial"}); err != nil {
		return err
	}
	if err := upsertModVersion(version.ProjectID, &version); err != nil {
		return err
	}
	return model.AddModToServer(s.ServerID, version.ProjectID, version.ID, strings.TrimPrefix(file.Path, "mods/"), false)
}

// One existing event-loop task handles all sessions; persist before eviction.
func PruneModpackSessions() error {
	modpackSessions.Lock()
	defer modpackSessions.Unlock()
	for id, s := range modpackSessions.items {
		select {
		case <-s.done:
		default:
			continue
		}
		if time.Since(s.EndedAt) < time.Hour {
			continue
		}
		if err := s.save(); err != nil {
			return err
		}
		delete(modpackSessions.items, id)
		modInstallEvents.mu.Lock()
		delete(modInstallEvents.streams, id)
		modInstallEvents.mu.Unlock()
	}
	return nil
}

func ShutdownModpackSessions() error {
	modpackSessions.Lock()
	modpackSessions.stopping = true
	sessions := make([]*ModpackSession, 0, len(modpackSessions.items))
	for _, s := range modpackSessions.items {
		s.cancel()
		sessions = append(sessions, s)
	}
	modpackSessions.Unlock()
	var errs []error
	for _, s := range sessions {
		<-s.done
		if err := s.save(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
