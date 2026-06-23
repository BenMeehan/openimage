# Openimage

Openimage is a terminal UI for generating AI images. Run it with no arguments
to open the interactive TUI — type a prompt, watch a live progress bar, and
see your image rendered inline. It also supports a CLI `generate` subcommand
for scripting and one-liners.

It supports thirteen providers — OpenAI, OpenRouter, Stability AI, Replicate,
Ideogram, DeepAI, GetImg.ai, Clipdrop, Segmind, Zhipu, Baidu, ByteDance, and
Alibaba — as well as any OpenAI-compatible endpoint.

openimage constructs the appropriate API request for the chosen provider,
authenticates with your credentials, downloads the resulting image, saves it
to disk, and renders it inline in the terminal. All parameters are
configurable via a YAML file or per-invocation flags.

---

## Installation

```bash
go install github.com/anomalyco/openimage@latest
```

Or from source:

```bash
git clone https://github.com/anomalyco/openimage.git
cd openimage
go build -o openimage .
```

---

## Quick start

```bash
openimage config set api-key sk-or-...
openimage
```

That opens the TUI. Type a prompt and press enter. The image generates with a
live progress bar and renders inline.

For CLI usage:

```bash
openimage generate "a watercolor painting of mountains at sunset"
openimage generate "a cat" --enhance
openimage generate "a fantasy landscape" --iterate
```

---

## Terminal UI

Running `openimage` with no arguments launches a full-screen interactive TUI:

```
┌─────────────────────────────────────────────────────┐
│                   ✧  openimage                      │
│                                                     │
│  describe the image you want to generate             │
│  ┌───────────────────────────────────────────────┐   │
│  │ > a cinematic portrait of a robot reading...  │   │
│  └───────────────────────────────────────────────┘   │
│                                                     │
│           enter  generate    ctrl+c  quit            │
└─────────────────────────────────────────────────────┘
```

Features:
- **Live progress bar** with gradient animation while generating
- **Inline image rendering** using Kitty, iTerm2, Sixel, or braille protocols
- **Dark theme** with accent colors and rounded borders
- Keyboard-driven: `enter` to generate, `ctrl+c` / `q` to quit

After generation, press `enter` to start a new prompt.

## CLI Usage

```
openimage generate <prompt> [flags]
```

```bash
openimage generate "a robot reading a book"
openimage generate -p "a robot reading a book"
openimage generate "cyberpunk city" --provider stability -m sd3
openimage generate "logo concept" -o logo.png -s 1024x1024 --open
openimage generate "variations" -n 4 --provider replicate -m black-forest-labs/flux-schnell
openimage generate "prompt" --provider openai --api-base https://api.together.ai/v1 -m black-forest-labs/FLUX.1-schnell
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--prompt` | `-p` | Text prompt |
| `--provider` | | Provider name |
| `--model` | `-m` | Model ID |
| `--size` | `-s` | Dimensions, e.g. `1024x1024` |
| `--quality` | `-q` | Quality or speed tier |
| `--style` | | Style preset |
| `--output` | `-o` | Output file path |
| `--count` | `-n` | Number of images (default 1) |
| `--enhance` | | Improve prompt with an LLM before generation |
| `--iterate` | `-i` | Interactive generation with feedback and refinement |
| `--api-base` | | Override API base URL |
| `--api-key` | | Override API key |
| `--open` | | Open result with system viewer |

When `--output` is omitted, files are saved to `save_dir` with a timestamp:
`openimage-20260123-143052.png`.

---

## Configuration

openimage reads `openimage.yaml` from the current working directory. Every
config value can be overridden by a command-line flag.

```yaml
api_key: sk-or-v1-abc123
provider: openrouter
model: openai/dall-e-3
size: 1024x1024
save_dir: ~/Pictures/openimage
display: true
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `api_key` | string | — | API key |
| `api_base` | string | — | Override the default API base URL |
| `provider` | string | `openrouter` | Provider name |
| `model` | string | `openai/dall-e-3` | Model ID |
| `size` | string | `1024x1024` | Image dimensions |
| `quality` | string | — | Quality or speed tier (provider-specific) |
| `style` | string | — | Style preset (provider-specific) |
| `save_dir` | string | `.` | Output directory (supports `~`) |
| `display` | bool | `true` | Show images in the terminal |

Manage config with subcommands:

```
openimage config set <key> <value>
openimage config get <key>
openimage config list
openimage config path
```

### API key

Resolved in this order:

1. `--api-key` flag
2. `OPENIMAGE_API_KEY` env var
3. `OPENROUTER_API_KEY` env var (compatible with the OpenRouter CLI)
4. `api_key` in `openimage.yaml`

---

## Providers

| Name | Docs |
|------|------|
| `openrouter` | [docs](https://openrouter.ai/docs) |
| `openai` | [docs](https://platform.openai.com/docs) |
| `stability` | [docs](https://platform.stability.ai/docs) |
| `replicate` | [docs](https://replicate.com/docs) |
| `ideogram` | [docs](https://docs.ideogram.ai) |
| `deepai` | [docs](https://deepai.org/docs) |
| `getimg` | [docs](https://docs.getimg.ai) |
| `clipdrop` | [docs](https://clipdrop.co/apis) |
| `segmind` | [docs](https://docs.segmind.com) |
| `zhipu` | [docs](https://open.bigmodel.cn/dev/api/image/cogview) |
| `baidu` | [docs](https://cloud.baidu.com/doc/WENXINWORKSHOP) |
| `doubao` | [docs](https://www.volcengine.com/docs/6791) |
| `dashscope` | [docs](https://help.aliyun.com/zh/model-studio) |

Any OpenAI-compatible endpoint works via `--provider openai --api-base <url>`.

---

## Models

Model IDs are provider-specific. The table below lists some of the most popular
ones. For each provider's full catalog, see the docs linked above.

| Model | Provider | Notes |
|-------|----------|-------|
| `openai/dall-e-3` | openrouter, openai | 1024x1024, 1024x1792, 1792x1024 |
| `openai/dall-e-2` | openrouter, openai | 256x256, 512x512, 1024x1024 |
| `stability-ai/stable-diffusion-3` | openrouter | SD3 via OpenRouter |
| `stability-ai/stable-diffusion-xl` | openrouter | SDXL via OpenRouter |
| `black-forest-labs/flux-schnell` | openrouter, replicate | Fastest Flux |
| `black-forest-labs/flux-dev` | openrouter, replicate | Flux Dev |
| `black-forest-labs/flux-pro` | openrouter, replicate | Flux Pro |
| `bytedance/seedream-4.0` | openrouter | Seedream 4 via OpenRouter |
| `ideogram-ai/ideogram-v2` | openrouter | Ideogram v2 via OpenRouter |
| `google/gemini-2.5-flash-image` | openrouter | Gemini 2.5 Flash, a.k.a. Nano Banana |
| `google/gemini-3.1-flash-image` | openrouter | Nano Banana 2 — Gemini 3.1 Flash Image |
| `google/gemini-3-pro-image` | openrouter | Nano Banana Pro — Gemini 3 Pro Image |
| `google/gemini-3.1-flash-image-preview` | openrouter | Nano Banana 2 (preview) |
| `google/gemini-3-pro-image-preview` | openrouter | Nano Banana Pro (preview) |
| `openai/gpt-5-image` | openrouter | GPT-5 + GPT Image 1 |
| `openai/gpt-5-image-mini` | openrouter | GPT-5 Mini + GPT Image 1 Mini |
| `openai/gpt-5.4-image-2` | openrouter | GPT-5.4 + GPT Image 2 |
| `openrouter/auto` | openrouter | Auto-routes to best model |
| `gpt-image-2` | segmind | GPT-4o image generation |
| `flux-schnell` | segmind, getimg | Fast Flux |
| `flux-dev` | segmind | Flux Dev |
| `sdxl1.0-newreality-lightning` | segmind | Fast SDXL |
| `juggernaut-xl` | segmind | Photorealistic SDXL |
| `realistic-vision-v5` | segmind | Photorealistic |
| `dreamshaper-xl` | segmind | Artistic SDXL |
| `nano-banana-2` | segmind | Fast general-purpose |
| `sd3` | stability | Stable Diffusion 3 |
| `sd3.5-large-turbo` | segmind | SD3.5 Turbo |
| `core` | stability | Stable Image Core |
| `ultra` | stability | Stable Image Ultra |
| `v4` | ideogram | Latest, excellent text rendering |
| `v3` | ideogram | Previous version |
| `seedream-5-lite` | getimg | Fast and capable |
| `seedream-4.0` | getimg | Previous version |
| `stable-diffusion-xl` | getimg | SDXL |
| `cogview-3-plus` | zhipu | Highest quality CogView |
| `cogview-3-flash` | zhipu | Fast CogView |
| `cogview-4` | zhipu | Latest, multi-reference |
| `sd_xl` | baidu | ERNIE-ViLG SDXL |
| `doubao-seedream-3.0-t2i-250415` | doubao | Seedream 3.0 (ByteDance) |
| `wan2.6-t2i` | dashscope | Alibaba Wanxiang 2.6 |
| `wan2.5-t2i-preview` | dashscope | Alibaba Wanxiang 2.5 |
| `kling/kling-v3-image-generation` | dashscope | Kling V3 (Kuaishou) |

---

## Terminal display

openimage renders generated images directly in your terminal using the best
available protocol, detected at runtime:

| Protocol | Terminals |
|----------|-----------|
| **Kitty** | Kitty |
| **iTerm2** | iTerm2, WezTerm, VS Code |
| **Sixel** | xterm, foot, contour, tmux (with sixel enabled) |
| **Braille** | All terminals — bash, zsh, etc. (Unicode braille characters) |

The protocol is automatically selected based on your terminal environment
(`$TERM`, `$TERM_PROGRAM`, `$KITTY_WINDOW_ID`). No configuration needed.

Disable inline display with `openimage config set display false`.

---

## Development

Requires Go 1.22+.

```bash
git clone https://github.com/anomalyco/openimage.git
cd openimage
go mod tidy
go run .
go vet ./...
go build -o openimage .
```

```
cmd/
  root.go        Root command — launches TUI by default
  generate.go    generate subcommand (CLI)
  config.go      config subcommand
  tui.go         Terminal UI (bubbletea)
pkg/
  config/        YAML config read/write
  display/       Terminal image rendering (Kitty, iTerm2, Sixel, braille)
  provider/      Provider interface and implementations
main.go          Entry point
```

### Adding a provider

1. Create `pkg/provider/<name>.go` implementing the `Provider` interface
2. Register it in `New()` in `pkg/provider/provider.go`
3. Add it to the help text in `cmd/config.go` and `cmd/generate.go`

Keep provider files self-contained, follow existing error handling patterns,
and avoid new dependencies unless necessary.

---

## Contributing

Bug reports, feature requests, and pull requests are welcome. Fork the repo,
create a branch, run `go vet ./...`, and open a PR.

---

## License

MIT — see [LICENSE.md](LICENSE.md).
