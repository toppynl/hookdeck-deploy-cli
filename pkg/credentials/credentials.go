package credentials

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Credentials holds the resolved API key and optional project ID.
type Credentials struct {
	APIKey    string
	ProjectID string
}

// Resolve finds credentials using this priority:
//  1. HOOKDECK_API_KEY environment variable
//  2. Named profile from ~/.config/hookdeck/config.toml
//  3. Default profile from config.toml
func Resolve(profileName string) (*Credentials, error) {
	if key := os.Getenv("HOOKDECK_API_KEY"); key != "" {
		return &Credentials{APIKey: key}, nil
	}

	configPath := getConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("no credentials found: set HOOKDECK_API_KEY or run 'hookdeck login'")
	}

	creds, err := loadFromTOML(configPath, profileName)
	if err != nil {
		return nil, err
	}
	if creds.APIKey == "" {
		return nil, fmt.Errorf("no API key found in profile '%s' at %s", profileName, configPath)
	}
	return creds, nil
}

func getConfigPath() string {
	if _, err := os.Stat(".hookdeck/config.toml"); err == nil {
		return ".hookdeck/config.toml"
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	globalPath := filepath.Join(home, ".config", "hookdeck", "config.toml")
	if _, err := os.Stat(globalPath); err == nil {
		return globalPath
	}
	return ""
}

func loadFromTOML(path string, profileName string) (*Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if profileName == "" {
		if p, ok := raw["profile"].(string); ok {
			profileName = p
		} else {
			profileName = "default"
		}
	}

	section, ok := raw[profileName]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found in %s", profileName, path)
	}

	profileMap, ok := section.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("profile '%s' is not a valid section", profileName)
	}

	creds := &Credentials{}
	if key, ok := profileMap["api_key"].(string); ok {
		creds.APIKey = key
	}
	if pid, ok := profileMap["project_id"].(string); ok {
		creds.ProjectID = pid
	}
	return creds, nil
}
