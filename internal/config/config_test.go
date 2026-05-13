package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("K8S_DOC_LAB_NAME", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LabName != "k8s-doc-lab" {
		t.Fatalf("LabName = %q, want k8s-doc-lab", cfg.LabName)
	}
	if cfg.SnapChannel != "latest/stable" {
		t.Fatalf("SnapChannel = %q, want latest/stable", cfg.SnapChannel)
	}
	if cfg.StateDir != ".state" {
		t.Fatalf("StateDir = %q, want .state", cfg.StateDir)
	}
	if len(cfg.LXDProfiles) != 2 || cfg.LXDProfiles[0] != "default" || cfg.LXDProfiles[1] != "k8s-integration" {
		t.Fatalf("LXDProfiles = %v, want [default k8s-integration]", cfg.LXDProfiles)
	}
}

func TestLoadEnvironmentOverrides(t *testing.T) {
	t.Setenv("K8S_DOC_LAB_NAME", "demo")
	t.Setenv("K8S_DOC_LXD_REMOTE", "remote1")
	t.Setenv("K8S_DOC_LXD_IMAGE", "ubuntu:24.04")
	t.Setenv("K8S_DOC_LXD_PROFILES", "custom-profile, k8s-profile")
	t.Setenv("K8S_DOC_SNAP_CHANNEL", "latest/edge")
	t.Setenv("K8S_DOC_STATE_DIR", "/tmp/k8s-doc-state")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LabName != "demo" || cfg.LXDRemote != "remote1" || cfg.LXDImage != "ubuntu:24.04" || len(cfg.LXDProfiles) != 2 || cfg.LXDProfiles[0] != "custom-profile" || cfg.LXDProfiles[1] != "k8s-profile" || cfg.SnapChannel != "latest/edge" || cfg.StateDir != "/tmp/k8s-doc-state" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
