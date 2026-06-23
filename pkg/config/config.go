package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	APIKey   string `yaml:"api_key"`
	APIBase  string `yaml:"api_base"`
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Size     string `yaml:"size"`
	Quality  string `yaml:"quality"`
	Style    string `yaml:"style"`
	SaveDir  string `yaml:"save_dir"`
	Display  bool   `yaml:"display"`
}

func DefaultConfig() *Config {
	return &Config{
		Model:   "openai/dall-e-3",
		Size:    "1024x1024",
		SaveDir: "",
		Display: true,
	}
}

func ConfigPath() (string, error) {
	return "openimage.yaml", nil
}

func Load() (*Config, error) {
	cfgPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	cfgPath, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	if err := os.WriteFile(cfgPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func ResolveAPIKey(flagValue string, cfg *Config) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("OPENIMAGE_API_KEY"); env != "" {
		return env
	}
	if env := os.Getenv("OPENROUTER_API_KEY"); env != "" {
		return env
	}
	if cfg.APIKey != "" {
		return cfg.APIKey
	}
	return ""
}
