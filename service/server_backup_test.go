package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeBackupFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestBackupNoWorld(t *testing.T) {
	dir := t.TempDir()
	sm := &ServerManager{}
	if err := sm.BackUp("s", dir); !errors.Is(err, ErrNoWorld) {
		t.Fatalf("err=%v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("created files: %v err=%v", entries, err)
	}
}

func TestBackupCopyFailureCleansPartialBackup(t *testing.T) {
	dir := t.TempDir()
	writeBackupFile(t, filepath.Join(dir, "world", "level.dat"), "world")
	if err := os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "world", "z-broken")); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(dir, "backup", "20200101_000000", "level.dat")
	writeBackupFile(t, old, "older backup")
	sm := &ServerManager{}
	if err := sm.BackUp("s", dir); err == nil {
		t.Fatal("expected copy failure")
	}
	entries, err := os.ReadDir(filepath.Join(dir, "backup"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "20200101_000000" {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	data, err := os.ReadFile(old)
	if err != nil || string(data) != "older backup" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestBackupNameCollisionKeepsExistingBackup(t *testing.T) {
	dir := t.TempDir()
	writeBackupFile(t, filepath.Join(dir, "world", "level.dat"), "new world")
	now := time.Now()
	for _, stamp := range []time.Time{now, now.Add(time.Second)} {
		writeBackupFile(t, filepath.Join(dir, "backup", stamp.Format("20060102_150405"), "level.dat"), "previous backup")
	}
	sm := &ServerManager{}
	if err := sm.BackUp("s", dir); !errors.Is(err, os.ErrExist) {
		if time.Since(now) >= time.Second {
			t.Skip("clock moved beyond the pre-created backup names")
		}
		t.Fatalf("err=%v", err)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "backup"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(dir, "backup", entry.Name(), "level.dat"))
		if err != nil || string(data) != "previous backup" {
			t.Fatalf("data=%q err=%v", data, err)
		}
	}
}

func TestRecoverPreservesWorldOnInvalidBackupOrCopyFailure(t *testing.T) {
	for _, kind := range []string{"empty", "level directory", "copy failure", "success"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			writeBackupFile(t, filepath.Join(dir, "world", "level.dat"), "original world")
			writeBackupFile(t, filepath.Join(dir, "world", "old-only"), "original")
			backup := filepath.Join(dir, "backup", "20261114_120000")
			if err := os.MkdirAll(backup, 0700); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "level directory":
				if err := os.Mkdir(filepath.Join(backup, "level.dat"), 0700); err != nil {
					t.Fatal(err)
				}
			case "copy failure", "success":
				writeBackupFile(t, filepath.Join(backup, "level.dat"), "saved world")
				if kind == "copy failure" {
					if err := os.Symlink(filepath.Join(dir, "missing"), filepath.Join(backup, "z-broken")); err != nil {
						t.Fatal(err)
					}
				}
			}
			sm := &ServerManager{}
			err := sm.ServerSaveRollBack("s", "20261114_120000", dir)
			want := "original world"
			if kind == "success" {
				if err != nil {
					t.Fatal(err)
				}
				want = "saved world"
				if _, err := os.Stat(filepath.Join(dir, "world", "old-only")); !os.IsNotExist(err) {
					t.Fatalf("old file remains: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("expected restore failure")
				}
				if kind != "copy failure" && !errors.Is(err, ErrInvalidBackup) {
					t.Fatalf("err=%v", err)
				}
				if data, err := os.ReadFile(filepath.Join(dir, "world", "old-only")); err != nil || string(data) != "original" {
					t.Fatalf("old data=%q err=%v", data, err)
				}
			}
			data, err := os.ReadFile(filepath.Join(dir, "world", "level.dat"))
			if err != nil || string(data) != want {
				t.Fatalf("world=%q err=%v", data, err)
			}
			stages, err := filepath.Glob(filepath.Join(dir, ".recover-*"))
			if err != nil || len(stages) != 0 {
				t.Fatalf("stages=%v err=%v", stages, err)
			}
		})
	}
}

func TestRecoverRejectsSymlinkAndNonDirectory(t *testing.T) {
	for _, kind := range []string{"symlink", "file", "backup symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			outside := t.TempDir()
			writeBackupFile(t, filepath.Join(outside, "20261114_120000", "level.dat"), "outside world")
			backup := filepath.Join(dir, "backup")
			if kind == "backup symlink" {
				if err := os.Symlink(outside, backup); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Mkdir(backup, 0700); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(backup, "20261114_120000")
				if kind == "symlink" {
					if err := os.Symlink(filepath.Join(outside, "20261114_120000"), target); err != nil {
						t.Fatal(err)
					}
				} else {
					writeBackupFile(t, target, "file")
				}
			}
			sm := &ServerManager{}
			if err := sm.ServerSaveRollBack("s", "20261114_120000", dir); !errors.Is(err, ErrInvalidBackupName) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestListBackups(t *testing.T) {
	sm := &ServerManager{}
	svc := NewServerService(sm)
	dir := t.TempDir()
	backups, err := svc.ListBackups(dir)
	if err != nil || backups == nil || len(backups) != 0 {
		t.Fatalf("backups=%v err=%v", backups, err)
	}
	writeBackupFile(t, filepath.Join(dir, "backup", "20261114_120000", "level.dat"), "backup")
	backups, err = svc.ListBackups(dir)
	if err != nil || len(backups) != 1 || backups[0] != "20261114_120000" {
		t.Fatalf("backups=%v err=%v", backups, err)
	}
	broken := t.TempDir()
	writeBackupFile(t, filepath.Join(broken, "backup"), "not a directory")
	if _, err := svc.ListBackups(broken); err == nil {
		t.Fatal("expected filesystem error")
	}
}

func TestRecoverInvalidNameWhileRunning(t *testing.T) {
	sm := &ServerManager{servers: map[string]*Server{"s": {running: true}}}
	for _, name := range []string{"../x", ""} {
		if err := sm.ServerSaveRollBack("s", name, t.TempDir()); !errors.Is(err, ErrInvalidBackupName) {
			t.Fatalf("name=%q err=%v", name, err)
		}
	}
}
