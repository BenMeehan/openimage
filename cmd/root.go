package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anomalyco/openimage/pkg/config"
)

var (
	cfg     *config.Config
	apiKey  string
)

var rootCmd = &cobra.Command{
	Use:   "openimage",
	Short: "Generate AI images from the terminal",
	Long: `openimage is a CLI tool for generating images using AI models via OpenRouter.
Configure your API key and generate images directly from the terminal with
support for terminal-native image display.`,
	SilenceUsage: true,
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
