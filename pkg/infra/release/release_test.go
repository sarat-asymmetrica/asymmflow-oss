package release

import "testing"

func TestLoadManifest(t *testing.T) {
	manifest, err := LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if manifest.Product != "AsymmFlow" {
		t.Fatalf("product = %q", manifest.Product)
	}
	if manifest.Version != "0.1.0-alpha.1" {
		t.Fatalf("version = %q", manifest.Version)
	}
}

func TestCurrentUsesLdflagOverrides(t *testing.T) {
	oldVersion, oldCommit, oldBuildTime, oldDirty := Version, Commit, BuildTime, Dirty
	defer func() {
		Version, Commit, BuildTime, Dirty = oldVersion, oldCommit, oldBuildTime, oldDirty
	}()

	Version = "0.1.0-beta.1"
	Commit = "abc123"
	BuildTime = "2026-05-08T12:00:00Z"
	Dirty = "true"

	info := Current()
	if info.Version != "0.1.0-beta.1" {
		t.Fatalf("version = %q", info.Version)
	}
	if info.GitCommit != "abc123" {
		t.Fatalf("commit = %q", info.GitCommit)
	}
	if !info.Dirty {
		t.Fatal("dirty should be true")
	}
}
