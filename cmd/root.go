package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anomalyco/openimage/pkg/config"
	"github.com/anomalyco/openimage/pkg/provider"
)

var (
	cfg    *config.Config
	apiKey string
)

var rootCmd = &cobra.Command{
	Use:   "openimage",
	Short: "Generate AI images from the terminal",
	Long: `openimage is a CLI tool for generating images using AI models.
Configure your API key and generate images interactively or via the command line.`,
	Run: runRoot,
}

func runRoot(cmd *cobra.Command, args []string) {
	key := config.ResolveAPIKey(apiKey, cfg)
	if key == "" {
		fmt.Fprintln(os.Stderr, "API key required. Set via --api-key, OPENIMAGE_API_KEY, or openimage config set api-key <key>")
		os.Exit(1)
	}

	providerName := cfg.Provider
	if providerName == "" {
		providerName = "openrouter"
	}
	apiBase := cfg.APIBase

	p, err := provider.New(providerName, key, apiBase)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := runTUI(p, key, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for OpenRouter (overrides env OPENIMAGE_API_KEY)")
}
