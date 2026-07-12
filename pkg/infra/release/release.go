package release

import (
	"embed"
	"encoding/json"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// These variables can be overridden with -ldflags during release builds.
var (
	Version   string
	Commit    string
	BuildTime string
	Dirty     string
)

//go:embed manifest.json
var manifestFS embed.FS

type Manifest struct {
	Product                 string `json:"product"`
	Version                 string `json:"version"`
	Channel                 string `json:"channel"`
	ReleaseDate             string `json:"release_date"`
	ReleaseName             string `json:"release_name"`
	MinimumSupportedVersion string `json:"minimum_supported_version"`
	SchemaVersion           string `json:"schema_version"`
	Notes                   string `json:"notes"`
}

type BuildInfo struct {
	Manifest
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	Dirty     bool   `json:"dirty"`
	GoVersion string `json:"go_version"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

func LoadManifest() (Manifest, error) {
	data, err := manifestFS.ReadFile("manifest.json")
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func Current() BuildInfo {
	manifest, err := LoadManifest()
	if err != nil {
		manifest = Manifest{Product: "AsymmFlow", Version: "0.1.0-alpha.1", Channel: "alpha"}
	}

	info := BuildInfo{
		Manifest:  manifest,
		GitCommit: valueOrDefault(Commit, vcsRevision()),
		BuildTime: valueOrDefault(BuildTime, time.Now().UTC().Format(time.RFC3339)),
		Dirty:     parseDirty(Dirty) || vcsDirty(),
		GoVersion: runtime.Version(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}
	if Version != "" {
		info.Version = Version
	}
	return info
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "dev"
}

func parseDirty(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "dirty":
		return true
	default:
		return false
	}
}

func vcsRevision() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	for _, setting := range buildInfo.Settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			return setting.Value
		}
	}
	return "dev"
}

func vcsDirty() bool {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}
	for _, setting := range buildInfo.Settings {
		if setting.Key == "vcs.modified" {
			return setting.Value == "true"
		}
	}
	return false
}
