package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anomalyco/openimage/pkg/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage openimage configuration",
	Long:  `View and modify openimage configuration stored in openimage.yaml in the current directory.`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available keys:
  api-key    API key
  api-base   API base URL (overrides provider default)
  provider   API provider: openai, openrouter, stability, replicate, ideogram, deepai, getimg, clipdrop, segmind
  model      Model ID (e.g. openai/dall-e-3, sd3, black-forest-labs/flux-schnell, seedream-5-lite)
  size       Default image size (e.g. 1024x1024)
  quality    Quality or speed tier (provider-specific)
  style      Style preset (provider-specific)
  save-dir   Default save directory
  display    Whether to display images in terminal (true/false)`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the config file path",
	Args:  cobra.NoArgs,
	RunE:  runConfigPath,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Args:  cobra.NoArgs,
	RunE:  runConfigList,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configListCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	switch key {
	case "api-key":
		cfg.APIKey = value
	case "api-base":
		cfg.APIBase = value
	case "provider":
		cfg.Provider = value
	case "model":
		cfg.Model = value
	case "size":
		cfg.Size = value
	case "quality":
		cfg.Quality = value
	case "style":
		cfg.Style = value
	case "save-dir":
		cfg.SaveDir = value
	case "display":
		switch value {
		case "true":
			cfg.Display = true
		case "false":
			cfg.Display = false
		default:
			return fmt.Errorf("display must be 'true' or 'false'")
		}
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	switch key {
	case "api-key":
		fmt.Println(cfg.APIKey)
	case "api-base":
		fmt.Println(cfg.APIBase)
	case "provider":
		fmt.Println(cfg.Provider)
	case "model":
		fmt.Println(cfg.Model)
	case "size":
		fmt.Println(cfg.Size)
	case "quality":
		fmt.Println(cfg.Quality)
	case "style":
		fmt.Println(cfg.Style)
	case "save-dir":
		fmt.Println(cfg.SaveDir)
	case "display":
		fmt.Println(cfg.Display)
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	cfgPath, err := config.ConfigPath()
	if err != nil {
		return err
	}
	fmt.Println(cfgPath)
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	fmt.Printf("api-key:  %s\n", maskKey(cfg.APIKey))
	fmt.Printf("api-base: %s\n", cfg.APIBase)
	fmt.Printf("provider: %s\n", cfg.Provider)
	fmt.Printf("model:    %s\n", cfg.Model)
	fmt.Printf("size:    %s\n", cfg.Size)
	fmt.Printf("quality: %s\n", cfg.Quality)
	fmt.Printf("style:   %s\n", cfg.Style)
	fmt.Printf("save-dir: %s\n", cfg.SaveDir)
	fmt.Printf("display: %v\n", cfg.Display)
	cfgPath, err := config.ConfigPath()
	if err == nil {
		fmt.Fprintf(os.Stderr, "\nConfig file: %s\n", cfgPath)
	}
	return nil
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
