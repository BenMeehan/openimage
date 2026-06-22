package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/anomalyco/openimage/pkg/config"
	"github.com/anomalyco/openimage/pkg/display"
	"github.com/anomalyco/openimage/pkg/provider"
)

var (
	prompt       string
	model        string
	size         string
	quality      string
	style        string
	output       string
	apiBase      string
	providerName string
	openFile     bool
	n            int
)

var generateCmd = &cobra.Command{
	Use:   "generate [prompt]",
	Short: "Generate an image from a text prompt",
	Long: `Generate an AI image from a text prompt.

Supported providers: openai, openrouter, stability, replicate, ideogram,
deepai, getimg, clipdrop, segmind, and any OpenAI-compatible endpoint.
The prompt can be provided as a positional argument or via the --prompt flag.
The generated image is displayed in the terminal (if supported) and saved to disk.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Text prompt describing the image to generate")
	generateCmd.Flags().StringVar(&providerName, "provider", "", "API provider: openai, openrouter, stability (default from config)")
	generateCmd.Flags().StringVarP(&model, "model", "m", "", "Model to use (default from config)")
	generateCmd.Flags().StringVarP(&size, "size", "s", "", "Image size, e.g. 1024x1024, 1792x1024")
	generateCmd.Flags().StringVarP(&quality, "quality", "q", "", "Image quality: standard or hd (OpenAI) / style preset (Stability)")
	generateCmd.Flags().StringVar(&style, "style", "", "Image style: vivid or natural (OpenAI) / style preset (Stability)")
	generateCmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (auto-generated if not set)")
	generateCmd.Flags().StringVar(&apiBase, "api-base", "", "API base URL (overrides provider default)")
	generateCmd.Flags().BoolVar(&openFile, "open", false, "Open generated image with system viewer")
	generateCmd.Flags().IntVarP(&n, "count", "n", 1, "Number of images to generate")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	if len(args) > 0 && prompt == "" {
		prompt = args[0]
	}

	if prompt == "" {
		return fmt.Errorf("prompt is required; provide it as an argument or with --prompt")
	}

	key := config.ResolveAPIKey(apiKey, cfg)
	if key == "" {
		return fmt.Errorf(
			"API key is required. Set it via:\n"+
				"  - --api-key flag\n"+
				"  - OPENIMAGE_API_KEY environment variable\n"+
				"  - openimage config set api-key <key>",
		)
	}

	useModel := model
	if useModel == "" {
		useModel = cfg.Model
	}
	useSize := size
	if useSize == "" {
		useSize = cfg.Size
	}
	useQuality := quality
	if useQuality == "" {
		useQuality = cfg.Quality
	}
	useStyle := style
	if useStyle == "" {
		useStyle = cfg.Style
	}
	useProvider := providerName
	if useProvider == "" {
		useProvider = cfg.Provider
	}
	useAPIBase := apiBase
	if useAPIBase == "" {
		useAPIBase = cfg.APIBase
	}

	p, err := provider.New(useProvider, key, useAPIBase)
	if err != nil {
		return err
	}

	for i := 0; i < n; i++ {
		if n > 1 {
			fmt.Fprintf(os.Stderr, "Generating image %d/%d with %s...\n", i+1, n, p.Name())
		} else {
			fmt.Fprintf(os.Stderr, "Generating image with %s...\n", p.Name())
		}

		img, err := p.GenerateImage(&provider.GenerateParams{
			Prompt:  prompt,
			Model:   useModel,
			N:       1,
			Size:    useSize,
			Quality: useQuality,
			Style:   useStyle,
		})
		if err != nil {
			return fmt.Errorf("generating image: %w", err)
		}

		if img.RevisedPrompt != "" {
			fmt.Fprintf(os.Stderr, "Revised prompt: %s\n", img.RevisedPrompt)
		}

		outPath := output
		if outPath == "" {
			outPath = resolveOutputPath(i, n)
		} else if n > 1 && output != "" {
			ext := filepath.Ext(outPath)
			base := outPath[:len(outPath)-len(ext)]
			outPath = fmt.Sprintf("%s_%d%s", base, i+1, ext)
		}

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}

		if err := os.WriteFile(outPath, img.Data, 0644); err != nil {
			return fmt.Errorf("writing image: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Saved: %s (%d bytes)\n", outPath, len(img.Data))

		if cfg.Display {
			display.ShowImage(img.Data, outPath)
		}

		if openFile {
			openWithSystemViewer(outPath)
		}
	}

	return nil
}

func resolveOutputPath(index, total int) string {
	dir := cfg.SaveDir
	if dir == "" {
		dir = "."
	}

	home, _ := os.UserHomeDir()
	if len(dir) > 0 && dir[0] == '~' {
		dir = filepath.Join(home, dir[1:])
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = "."
	}

	ts := time.Now().Format("20060102-150405")
	if total > 1 {
		return filepath.Join(absDir, fmt.Sprintf("openimage-%s-%d.png", ts, index+1))
	}
	return filepath.Join(absDir, fmt.Sprintf("openimage-%s.png", ts))
}

func openWithSystemViewer(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default:
		return
	}
	_ = cmd.Start()
}
