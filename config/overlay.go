package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const OverlayConfigFileName = "overlay.yaml"

// OverlayEntry is a session-scoped extra project attached to a workspace.
type OverlayEntry struct {
	Name     string            `yaml:"name"`
	Path     string            `yaml:"path"`
	Metadata map[string]string `yaml:"metadata,omitempty"`
}

// OverlayConfig maps workspace name to at most one overlay project.
type OverlayConfig struct {
	Version  int                     `yaml:"version"`
	Overlays map[string]OverlayEntry `yaml:"overlays"`
}

func GetOverlayConfigPath() (string, error) {
	globalDir, err := GetGlobalConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(globalDir, OverlayConfigFileName), nil
}

func LoadOverlayConfig() (*OverlayConfig, error) {
	configPath, err := GetOverlayConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &OverlayConfig{Version: 1, Overlays: map[string]OverlayEntry{}}, nil
		}
		return nil, fmt.Errorf("failed to read overlay config: %w", err)
	}
	var cfg OverlayConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse overlay config: %w", err)
	}
	if cfg.Overlays == nil {
		cfg.Overlays = map[string]OverlayEntry{}
	}
	return &cfg, nil
}

func SaveOverlayConfig(cfg *OverlayConfig) error {
	globalDir, err := GetGlobalConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return fmt.Errorf("failed to create global config directory: %w", err)
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Overlays == nil {
		cfg.Overlays = map[string]OverlayEntry{}
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal overlay config: %w", err)
	}
	configPath := filepath.Join(globalDir, OverlayConfigFileName)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write overlay config: %w", err)
	}
	return nil
}

func (c *OverlayConfig) Get(workspaceName string) (OverlayEntry, bool) {
	if c == nil || c.Overlays == nil {
		return OverlayEntry{}, false
	}
	entry, ok := c.Overlays[workspaceName]
	return entry, ok
}

func (c *OverlayConfig) Set(workspaceName string, entry OverlayEntry) {
	if c.Overlays == nil {
		c.Overlays = map[string]OverlayEntry{}
	}
	c.Overlays[workspaceName] = entry
}

func (c *OverlayConfig) Delete(workspaceName string) {
	if c == nil || c.Overlays == nil {
		return
	}
	delete(c.Overlays, workspaceName)
}

// MergeOverlay returns a copy of ws with the session overlay appended when present.
func MergeOverlay(ws *Workspace, overlay OverlayEntry, ok bool) *Workspace {
	if ws == nil {
		return nil
	}
	cp := *ws
	cp.Projects = append([]ProjectEntry(nil), ws.Projects...)
	if !ok || overlay.Path == "" {
		return &cp
	}
	name := overlay.Name
	if name == "" {
		name = "overlay"
	}
	for _, p := range cp.Projects {
		if p.Name == name || p.Path == overlay.Path {
			return &cp
		}
	}
	cp.Projects = append(cp.Projects, ProjectEntry{
		Name:     name,
		Path:     overlay.Path,
		Metadata: overlay.Metadata,
	})
	return &cp
}
