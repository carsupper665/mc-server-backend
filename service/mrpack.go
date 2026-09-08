package service

import (
	"archive/zip"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"

	"go-backend/common"
)

// Blacklist entries are instance-relative paths, directory prefixes ending in /,
// or path.Match patterns. env.server=unsupported is always excluded.
var DefaultMRPackBlacklist = "options.txt,optionsof.txt,servers.dat,servers.dat_old,resourcepacks/,shaderpacks/,screenshots/,config/*-client.*,config/iris.properties,config/sodium-options.json"

func MRPackLimit() int64 {
	return int64(common.GetEnvOrDefault("MRPACK_MAX_SIZE_MB", 150)) << 20
}

func MRPackBlacklist() []string {
	value, exists := os.LookupEnv("MRPACK_BLACKLIST")
	if !exists {
		value = DefaultMRPackBlacklist
	}
	return strings.Split(value, ",")
}

type MRPackFile struct {
	Path      string            `json:"path"`
	Hashes    map[string]string `json:"hashes"`
	Env       map[string]string `json:"env,omitempty"`
	Downloads []string          `json:"downloads"`
	FileSize  int64             `json:"fileSize"`
}

type MRPack struct {
	FormatVersion int               `json:"formatVersion"`
	Game          string            `json:"game"`
	VersionID     string            `json:"versionId"`
	Name          string            `json:"name"`
	Summary       string            `json:"summary,omitempty"`
	Dependencies  map[string]string `json:"dependencies"`
	Files         []MRPackFile      `json:"files"`
	Overrides     []*zip.File       `json:"-"`
	Skipped       []string          `json:"-"`
}

func validPackPath(name string) bool {
	if name == "" || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\:\x00") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." || part == "." || part == "" {
			return false
		}
	}
	return true
}

func packBlocked(name string, blacklist []string) bool {
	for _, pattern := range blacklist {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		matched, _ := path.Match(pattern, name)
		if matched || strings.HasSuffix(pattern, "/") && strings.HasPrefix(name, pattern) {
			return true
		}
	}
	return false
}

func packURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return fmt.Errorf("invalid HTTPS download: %s", raw)
	}
	for _, host := range strings.Split(common.GetEnvOrDefaultString("MRPACK_DOWNLOAD_HOSTS", "cdn.modrinth.com,github.com,raw.githubusercontent.com,gitlab.com,release-assets.githubusercontent.com,objects.githubusercontent.com"), ",") {
		if u.Hostname() == strings.TrimSpace(host) {
			return nil
		}
	}
	return fmt.Errorf("download host not allowed: %s", u.Hostname())
}

func ParseMRPack(reader io.ReaderAt, size int64, blacklist []string) (*MRPack, error) {
	if size <= 0 || size > MRPackLimit() {
		return nil, fmt.Errorf("mrpack exceeds %d MiB", MRPackLimit()>>20)
	}
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("invalid mrpack ZIP: %w", err)
	}
	var index *zip.File
	var expanded uint64
	for _, entry := range archive.File {
		if entry.UncompressedSize64 > uint64(MRPackLimit())-expanded {
			return nil, fmt.Errorf("expanded mrpack exceeds %d MiB", MRPackLimit()>>20)
		}
		expanded += entry.UncompressedSize64
		if entry.Name == "modrinth.index.json" {
			if index != nil {
				return nil, fmt.Errorf("duplicate modrinth.index.json")
			}
			index = entry
		}
	}
	if index == nil {
		return nil, fmt.Errorf("missing modrinth.index.json")
	}
	r, err := index.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var pack MRPack
	if err := json.NewDecoder(r).Decode(&pack); err != nil {
		return nil, fmt.Errorf("invalid modrinth.index.json: %w", err)
	}
	if pack.FormatVersion != 1 || pack.Game != "minecraft" || pack.Name == "" || pack.VersionID == "" || pack.Dependencies["minecraft"] == "" {
		return nil, fmt.Errorf("expected formatVersion 1, minecraft, name, versionId and minecraft dependency")
	}
	seen := map[string]bool{}
	files := pack.Files
	pack.Files = nil
	for _, file := range files {
		if !validPackPath(file.Path) {
			return nil, fmt.Errorf("invalid file path: %s", file.Path)
		}
		key := strings.ToLower(file.Path)
		if seen[key] {
			return nil, fmt.Errorf("duplicate file path: %s", file.Path)
		}
		seen[key] = true
		for side, value := range file.Env {
			if (side != "client" && side != "server") || (value != "required" && value != "optional" && value != "unsupported") {
				return nil, fmt.Errorf("invalid env for %s", file.Path)
			}
		}
		if file.Env["server"] == "unsupported" || packBlocked(file.Path, blacklist) {
			pack.Skipped = append(pack.Skipped, file.Path)
			continue
		}
		for algorithm, length := range map[string]int{"sha1": 20, "sha512": 64} {
			hash, err := hex.DecodeString(file.Hashes[algorithm])
			if err != nil || len(hash) != length {
				return nil, fmt.Errorf("invalid %s for %s", algorithm, file.Path)
			}
		}
		if len(file.Downloads) == 0 || file.FileSize < 0 {
			return nil, fmt.Errorf("missing downloads or invalid fileSize: %s", file.Path)
		}
		for _, raw := range file.Downloads {
			if err := packURL(raw); err != nil {
				return nil, err
			}
		}
		pack.Files = append(pack.Files, file)
	}
	// Apply the common layer first, independent of ZIP entry order.
	for _, prefix := range []string{"overrides/", "server-overrides/"} {
		seen = map[string]bool{}
		for _, entry := range archive.File {
			if entry.FileInfo().IsDir() || !strings.HasPrefix(entry.Name, prefix) {
				continue
			}
			name := strings.TrimPrefix(entry.Name, prefix)
			if !validPackPath(name) || !entry.Mode().IsRegular() {
				return nil, fmt.Errorf("invalid override: %s", entry.Name)
			}
			key := strings.ToLower(name)
			if seen[key] {
				return nil, fmt.Errorf("duplicate override: %s", entry.Name)
			}
			seen[key] = true
			if packBlocked(name, blacklist) {
				pack.Skipped = append(pack.Skipped, entry.Name)
				continue
			}
			pack.Overrides = append(pack.Overrides, entry)
		}
	}
	return &pack, nil
}

func (p *MRPack) Loader() (string, string, error) {
	loader, version := Vanilla, ""
	for id, value := range p.Dependencies {
		if id == "minecraft" {
			continue
		}
		if id != "fabric-loader" && id != "quilt-loader" && id != "forge" && id != "neoforge" {
			return "", "", fmt.Errorf("unsupported dependency: %s", id)
		}
		if loader != Vanilla || value == "" {
			return "", "", fmt.Errorf("expected one loader with a version")
		}
		loader, version = strings.TrimSuffix(id, "-loader"), value
	}
	return loader, version, nil
}
