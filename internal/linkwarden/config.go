package linkwarden

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// File holds credentials and defaults; token should be stored with mode 0600.
type File struct {
	URL        string `json:"url"`
	Token      string `json:"token"`
	AllowWrite bool   `json:"allow_write"`
}

type Config struct {
	URL              string
	Token            string
	AllowWrite       bool
	AllowWriteSource string
	Source           string
}

func LoadConfig(getenv func(string) string) (Config, error) {
	cfg := Config{}
	if v := getenv("LINKWARDEN_URL"); v != "" {
		cfg.URL = v
		cfg.Source = "LINKWARDEN_URL"
	}
	if v := getenv("LINKWARDEN_TOKEN"); v != "" {
		cfg.Token = v
		if cfg.Source == "" {
			cfg.Source = "LINKWARDEN_TOKEN"
		}
	}
	for _, path := range configPaths(getenv) {
		f, err := readConfigFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return Config{}, err
		}
		auditConfigFilePerm(path)
		if cfg.URL == "" && f.URL != "" {
			cfg.URL = f.URL
			if cfg.Source == "" {
				cfg.Source = path
			}
		}
		if cfg.Token == "" && f.Token != "" {
			cfg.Token = f.Token
			if cfg.Source == "" {
				cfg.Source = path
			}
		}
		if f.AllowWrite {
			cfg.AllowWrite = true
			if cfg.AllowWriteSource == "" {
				cfg.AllowWriteSource = path
			}
		}
	}
	allowed, src := EffectiveAllowWrite(getenv, cfg.AllowWrite, cfg.AllowWriteSource)
	cfg.AllowWrite = allowed
	if src != "" {
		cfg.AllowWriteSource = src
	}
	if cfg.URL != "" {
		n, _, err := ValidateBaseURL(cfg.URL)
		if err != nil {
			return Config{}, err
		}
		cfg.URL = n
	}
	return cfg, nil
}

func configPaths(getenv func(string) string) []string {
	var paths []string
	if p := getenv("LINKWARDEN_CONFIG"); p != "" {
		paths = append(paths, p)
	}
	if dir, err := configDirFrom(getenv); err == nil {
		paths = append(paths, filepath.Join(dir, "config.json"))
	}
	return paths
}

func readConfigFile(path string) (File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, err
	}
	return f, nil
}

func ConfigDir() (string, error) {
	return configDirFrom(os.Getenv)
}

func configDirFrom(getenv func(string) string) (string, error) {
	if v := getenv("LINKWARDEN_CONFIG_DIR"); v != "" {
		return v, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "linkwarden"), nil
}

// WriteAllowWriteFile stores allow_write opt-in (mode 0600, parent dir 0700).
func WriteAllowWriteFile(enabled bool) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	val := "false"
	if enabled {
		val = "true"
	}
	return os.WriteFile(filepath.Join(dir, "allow_write"), []byte(val), 0o600)
}

// WriteConfigFile writes config JSON with mode 0600 (parent dir 0700).
func WriteConfigFile(path string, f File) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o600)
}

func bytesTrimSpace(b []byte) []byte {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\n' || b[0] == '\r' || b[0] == '\t') {
		b = b[1:]
	}
	for len(b) > 0 {
		c := b[len(b)-1]
		if c != ' ' && c != '\n' && c != '\r' && c != '\t' {
			break
		}
		b = b[:len(b)-1]
	}
	return b
}
