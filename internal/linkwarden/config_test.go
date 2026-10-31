package linkwarden

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEffectiveAllowWrite(t *testing.T) {
	if allowed, _ := EffectiveAllowWrite(func(string) string { return "true" }, false, ""); !allowed {
		t.Fatal("env true")
	}
	if allowed, src := EffectiveAllowWrite(func(string) string { return "" }, true, "/cfg"); !allowed || src != "/cfg" {
		t.Fatalf("config %v %q", allowed, src)
	}
	if allowed, _ := EffectiveAllowWrite(func(string) string { return "" }, false, ""); allowed {
		t.Fatal("default read")
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	getenv := func(k string) string {
		switch k {
		case "LINKWARDEN_URL":
			return "https://marks.example"
		case "LINKWARDEN_TOKEN":
			return "tok"
		default:
			return ""
		}
	}
	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.URL != "https://marks.example" || cfg.Token != "tok" || cfg.AllowWrite {
		t.Fatalf("%+v", cfg)
	}
}

func TestLoadConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := WriteConfigFile(path, File{URL: "https://x", Token: "t", AllowWrite: true}); err != nil {
		t.Fatal(err)
	}
	getenv := func(k string) string {
		if k == "LINKWARDEN_CONFIG" {
			return path
		}
		return ""
	}
	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowWrite || cfg.URL != "https://x" {
		t.Fatalf("%+v", cfg)
	}
}

func TestWriteAllowWriteFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LINKWARDEN_CONFIG_DIR", dir)
	if err := WriteAllowWriteFile(true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "allow_write"))
	if err != nil || string(b) != "true" {
		t.Fatalf("%q %v", b, err)
	}
}
