//service/serverManager.go

package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go-backend/common"
	"go-backend/model"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidBackup      = errors.New("invalid backup")
	ErrNoWorld            = errors.New("伺服器尚未產生世界，請先啟動一次")
	ErrInvalidBackupName  = errors.New("invalid backup name")
	ErrAlreadyRunning     = errors.New("server already running")
	ErrNotFound           = errors.New("server Not Found")
	ErrMaxReached         = errors.New("user has reached the maximum number of servers")
	ErrServerRunning      = errors.New("cannot Backup while server is running")
	ErrAlreadyDisabled    = errors.New("mod is already disabled")
	ErrAlreadyEnable      = errors.New("mod is already enable")
	ErrServerBusy         = errors.New("server is busy")
	ErrModMetadataMissing = errors.New("mod file metadata missing")
)

type Server struct {
	sid       string
	oid       string
	workDir   string
	maxMem    string
	minMem    string
	port      string
	cmd       *exec.Cmd
	Monitor   *Monitor
	stdin     io.Writer
	stdout    io.Reader
	logBuffer *bytes.Buffer
	running   bool
	exp       time.Time
	sdc       func(string)
	args      []string
	mu        sync.RWMutex
}

func NewServer(sid, oid, workDir, maxMem, minMem string, portStr string, callback func(string), args []string) *Server {
	return &Server{
		sid:       sid,
		oid:       oid,
		workDir:   workDir,
		maxMem:    maxMem,
		minMem:    minMem,
		port:      portStr,
		running:   false,
		sdc:       callback,
		args:      args,
		logBuffer: &bytes.Buffer{},
	}
}

// SplitJVMArgs splits stored JVM args by whitespace (strings.Fields).
// Quotes are not special and remain part of the token.
func SplitJVMArgs(args string) []string {
	return strings.Fields(args)
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return ErrAlreadyRunning
	}
	// 建立命令參數
	cmdArgs := []string{
		"-Xms" + s.minMem,
		"-Xmx" + s.maxMem,
	}
	cmdArgs = append(cmdArgs, s.args...)
	cmdArgs = append(cmdArgs, "-jar", "server.jar", "--port", s.port)
	common.Logger.Debugf("Starting server with args: %v", cmdArgs)
	cmd := exec.CommandContext(context.Background(), "java", cmdArgs...)
	cmd.Dir = s.workDir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	s.cmd = cmd
	s.stdin = stdin
	s.stdout = stdout
	s.logBuffer.Reset()

	if err := cmd.Start(); err != nil {
		return err
	}
	s.running = true
	pid := int32(cmd.Process.Pid)
	mon, err := SetUpMonitor(pid)
	s.Monitor = mon
	s.Monitor.Start(2 * time.Second) // 2 Seconds interval
	s.exp = time.Now().Add(3 * time.Minute)
	go s.captureLogs()
	go s.waitAndCleanup()
	return nil
}

func (s *Server) captureLogs() {
	io.Copy(s.logBuffer, s.stdout)
}

func (s *Server) waitAndCleanup() {
	s.cmd.Wait()
	s.mu.Lock()
	s.running = false
	s.exp = time.Now().Add(3 * time.Minute)
	s.Monitor.Stop()
	s.mu.Unlock()
}

func (s *Server) GetLatestSnapshot() (snap Snapshot, ok bool) {
	// 先在鎖內只取需要的指標/狀態，避免鎖耦合
	s.mu.RLock()
	mon := s.Monitor
	status := s.running
	s.mu.RUnlock()

	if mon != nil {
		return mon.Snapshot(), true
	}
	if !status {
		return Snapshot{}, false
	}
	return Snapshot{}, false
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == nil || !s.running {
		return errors.New("server not running")
	}

	_, _ = io.WriteString(s.stdin, "stop\n")

	timeout := 30 * time.Second
	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		// 超時，強制 kill
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		<-done // 等候 goroutine 結束（可以忽略 error）
	case err := <-done:
		if err != nil {
			common.SysError(err.Error())
		}
	}

	s.running = false
	s.exp = time.Now().Add(3 * time.Minute)
	return nil
}

func (s *Server) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start()
}

func (s *Server) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return "stopped"
	}
	return "running"
}

func (s *Server) Port() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

func (s *Server) ID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sid
}

func (s *Server) ReadLatestLog() string {
	data := s.logBuffer.String()
	if len(data) > (1024 * 8) {
		s.mu.Lock()
		defer s.mu.Unlock()
		return data[len(data)-(1024*8):]
	}
	return data
}

// SendCommand 發送指令到伺服器 stdin
func (s *Server) SendCommand(cmd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return errors.New("server not running")
	}
	_, err := io.WriteString(s.stdin, cmd+"\n")
	return err
}

func (s *Server) SetProperty(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New("必須先停止 server 才能修改 server.properties")
	}
	return UpdateProperty(s.workDir, key, value)
}

func (s *Server) ShutDown() error {
	s.mu.RLock()
	status := s.running
	callback := s.sdc
	sid := s.sid
	s.mu.RUnlock()
	if status {
		if err := s.Stop(); err != nil {
			return err
		}
	}
	callback(sid)
	return nil
}

// ---------------- ServerManager ----------------

type ServerManager struct {
	servers        map[string]*Server
	availablePorts []int
	usingPorts     map[int]string //port -> server ID
	mu             sync.RWMutex
}

func NewServerManager(ports []int) *ServerManager {
	sm := &ServerManager{
		servers:        make(map[string]*Server),
		availablePorts: ports,
		usingPorts:     make(map[int]string),
	}
	go sm.cleanupExpired()
	return sm
}

func (sm *ServerManager) ServerSaveList(workDir string) ([]string, error) {
	// just read dir, lock not needed
	list := make([]string, 0)

	// os read dir
	backupDir := filepath.Join(workDir, "backup")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return list, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			list = append(list, entry.Name())
		}
	}

	return list, nil
}

func (sm *ServerManager) CleanUpHook() *common.ErrCleanUpStack {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	errStack := &common.ErrCleanUpStack{}

	for sid, srv := range sm.servers {
		s := srv.Status()
		if s == "stopped" {
			delete(sm.servers, sid)
			common.SysLog(fmt.Sprintf("Cleaned up server: %s", sid))
			continue
		}

		if err := srv.Stop(); err != nil {
			errStack.Add(sid, err)
			common.SysError(fmt.Sprintf("Failed to clean up server %s: %v", sid, err))
			continue
		}
		delete(sm.servers, sid)
		common.SysLog(fmt.Sprintf("Cleaned up server: %s", sid))
	}

	if errStack.HasError() {
		return errStack
	}

	return nil
}

var backupNamePattern = regexp.MustCompile(`^[0-9]{8}_[0-9]{6}$`)

func (sm *ServerManager) ServerSaveRollBack(sid, fileName, workDir string) error {
	if fileName == "" || fileName == "." || fileName == ".." || strings.ContainsAny(fileName, `/\`) || filepath.Base(fileName) != fileName || !backupNamePattern.MatchString(fileName) {
		return ErrInvalidBackupName
	}

	sm.mu.RLock()
	srv, exists := sm.servers[sid]
	sm.mu.RUnlock()

	if exists && srv.Status() == "running" {
		return ErrServerRunning
	}

	backupDir := filepath.Join(workDir, "backup")
	src := filepath.Join(backupDir, fileName)
	rel, err := filepath.Rel(backupDir, src)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ErrInvalidBackupName
	}
	// Do not let a symlink turn a single path segment into an external path.
	for _, path := range []string{backupDir, src} {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return ErrInvalidBackupName
		}
	}

	level, err := os.Lstat(filepath.Join(src, "level.dat"))
	if errors.Is(err, os.ErrNotExist) {
		return ErrInvalidBackup
	}
	if err != nil {
		return err
	}
	if !level.Mode().IsRegular() {
		return ErrInvalidBackup
	}

	dst := filepath.Join(workDir, "world")
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stage, err := os.MkdirTemp(workDir, ".recover-")
	if err != nil {
		return err
	}
	preserveStage := false
	defer func() {
		if !preserveStage {
			_ = os.RemoveAll(stage)
		}
	}()
	next := filepath.Join(stage, "world")
	if err := common.Copy(src, next); err != nil {
		return err
	}

	old := filepath.Join(stage, "old-world")
	hadWorld := true
	if err := os.Rename(dst, old); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		hadWorld = false
	}
	if err := os.Rename(next, dst); err != nil {
		if hadWorld {
			if restoreErr := os.Rename(old, dst); restoreErr != nil {
				// Keep the original world on disk if recovery itself fails.
				preserveStage = true
				return errors.Join(err, fmt.Errorf("original world retained at %s: %w", old, restoreErr))
			}
		}
		return err
	}
	return os.RemoveAll(stage)
}

func (sm *ServerManager) countByOwner(oid string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	count := 0
	for _, srv := range sm.servers {
		if srv.oid == oid && srv.Status() == "running" {
			count++
		}
	}
	return count
}

func (sm *ServerManager) allocatePort() (int, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if len(sm.availablePorts) == 0 {
		return 0, errors.New("no available ports")
	}
	port := sm.availablePorts[0]
	sm.availablePorts = sm.availablePorts[1:]
	sm.usingPorts[port] = ""
	return port, nil
}

// assignPortToServer 將 port 綁定給 sid
func (sm *ServerManager) assignPortToServer(port int, sid string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.usingPorts[port] = sid
}

// releasePort 釋放回 pool
func (sm *ServerManager) releasePort(portStr string) {
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.usingPorts, port)
	sm.availablePorts = append(sm.availablePorts, port)
}

func (sm *ServerManager) releasePortWithOutLock(portStr string) {
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	delete(sm.usingPorts, port)
	sm.availablePorts = append(sm.availablePorts, port)
}

func (sm *ServerManager) StartServer(sid, oid, workDir, maxMem, minMem string, args []string) (*Server, error) {
	sm.mu.Lock()
	if s, exists := sm.servers[sid]; exists {
		err := s.Start()
		if err != nil && errors.Is(err, ErrAlreadyRunning) {
			sm.mu.Unlock()
			common.SysDebug("server already running sid: " + sid)
			return nil, ErrAlreadyRunning
		} else if err != nil {
			panic("Unknow error:" + err.Error())
		}
		sm.mu.Unlock()
		common.SysDebug("Server is running: " + sid)
		return s, nil // Server Running successfully
	}
	sm.mu.Unlock()

	var allocatedPort int
	p, err := sm.allocatePort()
	if err != nil {
		return nil, err
	}
	allocatedPort = p
	portStr := fmt.Sprintf("%d", p)

	srv := NewServer(sid, oid, workDir, maxMem, minMem, portStr, sm.shutDownServerCallback, args)
	sm.assignPortToServer(allocatedPort, sid)

	sm.mu.Lock()
	sm.servers[sid] = srv
	sm.mu.Unlock()

	if err := srv.Start(); err != nil {
		sm.mu.Lock()
		delete(sm.servers, sid)
		sm.releasePort(portStr)
		sm.mu.Unlock()
		return nil, err
	}
	common.SysDebug("Server Start: " + sid)
	return srv, nil
}

func (sm *ServerManager) SendCommand(sid string, cmd string) error {
	sm.mu.RLock()
	srv, exists := sm.servers[sid]
	sm.mu.RUnlock()

	if !exists {
		return ErrNotFound
	}

	if err := srv.SendCommand(cmd); err != nil {
		return err
	}

	return nil
}

func (sm *ServerManager) StopServer(sid string) error {
	sm.mu.RLock()
	srv, exists := sm.servers[sid]
	sm.mu.RUnlock()
	if !exists {
		return ErrNotFound
	}

	if err := srv.Stop(); err != nil {
		return err
	}

	// 不能重複釋放PORT 必須等到過期回收 在釋放
	// var p int
	// fmt.Sscanf(srv.port, "%d", &p)
	// sm.releasePort(p)
	return nil
}

func (sm *ServerManager) RestartServer(sid string) error {
	sm.mu.RLock()
	srv, exists := sm.servers[sid]
	sm.mu.RUnlock()
	if !exists {
		return ErrNotFound
	}
	return srv.Restart()
}

func (sm *ServerManager) GetServerUsage(sid string) (Snapshot, error) {
	sm.mu.RLock()
	srv, exists := sm.servers[sid]
	sm.mu.RUnlock()
	if !exists {
		return Snapshot{}, ErrNotFound
	}
	snap, ok := srv.GetLatestSnapshot()
	if !ok {
		return snap, ErrNotFound
	}
	return snap, nil
}

func (sm *ServerManager) GetServerStatus(sid string) (string, error) {
	sm.mu.RLock()
	srv, exists := sm.servers[sid]
	sm.mu.RUnlock()
	if !exists {
		return "stopped", nil
	}
	return srv.Status(), nil
}

func (sm *ServerManager) shutDownServerCallback(sid string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// shut down server時必須釋放port
	srv := sm.servers[sid]
	sm.releasePortWithOutLock(srv.port)
	delete(sm.servers, sid)
}

func (sm *ServerManager) ReadLatestLog(sid string) (string, error) {
	sm.mu.RLock()
	srv, exists := sm.servers[sid]
	sm.mu.RUnlock()

	if !exists {
		return "", ErrNotFound
	}

	return srv.ReadLatestLog(), nil
}

func (sm *ServerManager) BackUp(sid, workDir string) error {
	sm.mu.RLock()
	srv, exists := sm.servers[sid]
	sm.mu.RUnlock()

	if exists && srv.Status() == "running" {
		return ErrServerRunning
	}

	src := filepath.Join(workDir, "world")
	info, err := os.Stat(src)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNoWorld
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrNoWorld
	}

	backupDir := filepath.Join(workDir, "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}
	dst := filepath.Join(backupDir, time.Now().Format("20060102_150405"))
	// Never overwrite (or clean up) an earlier backup from the same second.
	if err := os.Mkdir(dst, 0755); err != nil {
		return err
	}
	if err := common.Copy(src, dst); err != nil {
		return errors.Join(err, os.RemoveAll(dst))
	}
	return nil
}

func (sm *ServerManager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		sm.mu.Lock()
		for sid, srv := range sm.servers {
			s := srv.Status()
			isExp := srv.exp.Before(now)
			if s == "stopped" && isExp {
				sm.releasePortWithOutLock(srv.port)
				delete(sm.servers, sid)
				common.SysLog(fmt.Sprintf("Server: %s del, port: %s", sid, srv.port))
			}
		}
		sm.mu.Unlock()
	}
}

func (sm *ServerManager) cleanup(sid string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	srv, exists := sm.servers[sid]
	if exists {
		sm.releasePortWithOutLock(srv.port)
		delete(sm.servers, sid)
		common.SysLog(fmt.Sprintf("Server: %s del, port: %s", sid, srv.port))
	}
}

func (sm *ServerManager) isRunning(sid string) bool {
	sm.mu.RLock()
	srv, exists := sm.servers[sid]
	sm.mu.RUnlock()
	return exists && srv.Status() == "running"
}

func (sm *ServerManager) DeleteServer(sid, dir string) error {
	sm.mu.RLock()
	srv, exists := sm.servers[sid]
	sm.mu.RUnlock()

	if exists {
		if srv.Status() == "running" {
			return ErrServerRunning
		}
		sm.cleanup(sid)
	}

	// 刪除dir底下的檔案
	path := dir
	return os.RemoveAll(path)
}

func (sm *ServerManager) DisableMod(sid, modPath string) error {
	if running := sm.isRunning(sid); running {
		return ErrServerRunning
	}
	// rename file to disable
	sm.mu.Lock()
	srv, exists := sm.servers[sid]
	sm.mu.Unlock()
	if exists {
		if ok := srv.mu.TryRLock(); !ok {
			return ErrServerBusy
		}
		defer srv.mu.RUnlock()
	}
	ext := strings.ToLower(filepath.Ext(modPath)) // ".disable" vor ".jar"
	if ext == ".disable" {
		return ErrAlreadyDisabled // already disable
	}
	newPath := modPath + ".disable"
	if err := os.Rename(modPath, newPath); err != nil {
		return err
	}

	return nil
}
func (sm *ServerManager) EnableMod(sid, modPath string) error {
	if running := sm.isRunning(sid); running {
		return ErrServerRunning
	}

	// rename file to enable
	ext := strings.ToLower(filepath.Ext(modPath)) // ".disable" or ".jar"
	if ext == ".jar" {
		return ErrAlreadyEnable // already enabled
	}
	newName := strings.TrimSuffix(modPath, ext)
	if err := os.Rename(modPath, newName); err != nil {
		return err
	}

	return nil

}

func (sm *ServerManager) DeleteMod(sid, modPath string) error {
	if running := sm.isRunning(sid); running {
		return ErrServerRunning
	}

	sm.mu.Lock()
	srv, exists := sm.servers[sid]
	sm.mu.Unlock()
	if exists {
		if ok := srv.mu.TryRLock(); !ok {
			return ErrServerBusy
		}
		defer srv.mu.RUnlock()
	}

	if err := os.Remove(modPath); err != nil {
		return err
	}
	return nil
}

func (sm *ServerManager) UpdateMod(sid, modId, workDir, oldModFile, modLoader, gameVersion string, autoUpdate bool) error {
	if modId == "" || model.IsLocalModID(modId) {
		return model.ErrModMetadataUnavailable
	}
	// TODO 類 transaction 機制 錯誤要負責復原所有東西 回到發生前
	if running := sm.isRunning(sid); running {
		return ErrServerRunning
	}
	sm.mu.Lock()
	srv, exists := sm.servers[sid]
	sm.mu.Unlock()
	if exists {
		if ok := srv.mu.TryRLock(); !ok {
			return ErrServerBusy
		}
		defer srv.mu.RUnlock()
	}

	oldModPath := oldModFile
	// rename old mod
	if err := os.Rename(oldModPath, oldModPath+".old"); err != nil {
		return err
	}

	isLatest, verId, err := model.IsUptodate(sid, modId)
	if err != nil {
		return err
	}

	if isLatest {
		_ = os.Rename(oldModPath+".old", oldModPath)
		return nil
	}

	newModInfo, err := getLatestOrSpecific(modId, modLoader, gameVersion, verId, false) // 先預設為false 此功能待測試
	if err != nil {
		return err
	}

	file, err := selectModFile(newModInfo.Files)
	if err != nil {
		return err
	}

	// download mod to work dir
	newFile, err := modDownload(*file, workDir)
	if err != nil {
		return err
	}

	//update db
	if err := model.AddModToServer(sid, modId, newModInfo.ID, file.Filename, autoUpdate); err != nil {
		_ = os.Remove(newFile)
		_ = os.Rename(oldModPath+".old", oldModPath)
		return err
	}
	_ = os.Remove(oldModPath + ".old")
	return nil
}
