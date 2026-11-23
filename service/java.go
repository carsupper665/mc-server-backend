package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// javaBuckets are the Java major versions that can be configured with JAVA_<major>.
var javaBuckets = []int{8, 17, 21, 25}

// javaEnvKey maps a required Java major to the smallest configurable bucket that can run it
// (e.g. 16 → JAVA_17). Versions newer than every bucket have no key.
func javaEnvKey(major int) string {
	for _, b := range javaBuckets {
		if major <= b {
			return fmt.Sprintf("JAVA_%d", b)
		}
	}
	return ""
}

// SelectJava returns the configured Java executable for a required major, or "java" from PATH.
func SelectJava(major int) string {
	if key := javaEnvKey(major); key != "" {
		if path := strings.TrimSpace(os.Getenv(key)); path != "" {
			return path
		}
	}
	return "java"
}

func javaConfigured() bool {
	for _, b := range javaBuckets {
		if strings.TrimSpace(os.Getenv(fmt.Sprintf("JAVA_%d", b))) != "" {
			return true
		}
	}
	return false
}

// FallbackJavaMajor approximates Mojang's javaVersion when version data is unavailable.
func FallbackJavaMajor(mc string) int {
	switch {
	case compareVersions(mc, "1.17") < 0:
		return 8
	case compareVersions(mc, "1.18") < 0:
		return 16
	case compareVersions(mc, "1.20.5") < 0:
		return 17
	case compareVersions(mc, "26") < 0:
		return 21
	}
	return 25
}

// RequiredJavaMajor reads javaVersion.majorVersion from Mojang's version data.
func RequiredJavaMajor(ctx context.Context, mc string) int {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if d, err := vanillaVersionDetail(ctx, mc); err == nil && d.JavaVersion.MajorVersion > 0 {
		return d.JavaVersion.MajorVersion
	}
	return FallbackJavaMajor(mc)
}

// JavaForMC picks the Java executable for launching or installing an MC version.
// Without any JAVA_* configured it is always "java" and no version data is fetched.
func JavaForMC(ctx context.Context, mc string) string {
	if !javaConfigured() || mc == "" {
		return "java"
	}
	return SelectJava(RequiredJavaMajor(ctx, mc))
}
