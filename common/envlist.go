package common

import (
	"os"
	"strings"
)

// EnvList reads a comma-separated env var, trimming spaces and dropping empty items.
func EnvList(key string) []string {
	return SplitList(os.Getenv(key))
}

// SplitList splits a comma-separated value, trimming spaces and dropping empty items.
func SplitList(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
