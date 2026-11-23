package service

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Launch modes stored per server (UserMinecraftServer.LaunchMode).
const (
	LaunchJar  = "jar"  // java ... -jar <LaunchFile>
	LaunchArgs = "args" // java ... @<LaunchFile> nogui (Forge 1.17+ / NeoForge)

	DefaultLaunchFile = "server.jar"
)

var ErrNoLaunchFile = errors.New("installer did not produce a launch file")

// Launch is everything needed to start a server process.
type Launch struct {
	Java    string
	Mode    string
	File    string
	MaxMem  string
	MinMem  string
	JVMArgs []string
}

// NormalizeLaunch fills defaults for servers created before launch modes were stored.
func NormalizeLaunch(mode, file string) (string, string) {
	if mode != LaunchArgs {
		mode = LaunchJar
	}
	if file == "" {
		if mode == LaunchArgs {
			return LaunchJar, DefaultLaunchFile
		}
		file = DefaultLaunchFile
	}
	return mode, file
}

// BuildLaunchArgs builds the java command-line arguments (without the executable).
func BuildLaunchArgs(l Launch) []string {
	mode, file := NormalizeLaunch(l.Mode, l.File)
	args := []string{"-Xms" + l.MinMem, "-Xmx" + l.MaxMem}
	args = append(args, l.JVMArgs...)
	if mode == LaunchArgs {
		return append(args, "@"+file, "nogui")
	}
	return append(args, "-jar", file)
}

func (l Launch) java() string {
	if l.Java == "" {
		return "java"
	}
	return l.Java
}

// DetectLaunch inspects an installed Forge/NeoForge directory.
// Modern installers write libraries/**/win_args.txt (unix_args.txt elsewhere);
// legacy Forge (1.12.2–1.16.5) writes forge-*.jar in the server root.
func DetectLaunch(workDir, goos string) (mode, file string, err error) {
	argsName := "unix_args.txt"
	if goos == "windows" {
		argsName = "win_args.txt"
	}
	var argsFiles []string
	libDir := filepath.Join(workDir, "libraries")
	if _, statErr := os.Stat(libDir); statErr == nil {
		walkErr := filepath.WalkDir(libDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == argsName {
				rel, relErr := filepath.Rel(workDir, path)
				if relErr != nil {
					return relErr
				}
				argsFiles = append(argsFiles, filepath.ToSlash(rel))
			}
			return nil
		})
		if walkErr != nil {
			return "", "", walkErr
		}
	}
	if len(argsFiles) > 0 {
		sort.Strings(argsFiles)
		// Prefer the loader's own args file over any other library that ships one.
		for _, f := range argsFiles {
			if strings.Contains(f, "/neoforge/") || strings.Contains(f, "/forge/") {
				return LaunchArgs, f, nil
			}
		}
		return LaunchArgs, argsFiles[0], nil
	}

	entries, err := os.ReadDir(workDir)
	if err != nil {
		return "", "", err
	}
	var jars []string
	for _, e := range entries {
		name := e.Name()
		lower := strings.ToLower(name)
		if e.IsDir() || !strings.HasPrefix(lower, "forge-") || !strings.HasSuffix(lower, ".jar") || strings.Contains(lower, "installer") {
			continue
		}
		jars = append(jars, name)
	}
	sort.Slice(jars, func(i, j int) bool {
		ui, uj := strings.Contains(strings.ToLower(jars[i]), "universal"), strings.Contains(strings.ToLower(jars[j]), "universal")
		if ui != uj {
			return !ui
		}
		return jars[i] < jars[j]
	})
	if len(jars) > 0 {
		return LaunchJar, jars[0], nil
	}
	return "", "", ErrNoLaunchFile
}
