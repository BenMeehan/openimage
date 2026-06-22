# openimage

Generate AI images from your terminal.

openimage is a single-binary CLI tool that talks to image generation APIs,
saves the result to disk, and renders the image directly in your terminal using
native graphics protocols. No external image viewer required.

It supports nine providers natively and any OpenAI-compatible endpoint — you
pick the model, set the size, and openimage handles the rest: API
authentication, parameter translation, response parsing, file I/O, and
terminal display.

---

## Why

AI image generation tools are overwhelmingly GUI-based web apps. Every
interaction requires context-switching out of the terminal, uploading a prompt
into a browser, waiting, and downloading a file. openimage puts the entire
workflow into a single command. It is designed to work the way CLI tools work:
composable, scriptable, and configurable.

If you spend your day in a terminal and occasionally need an AI-generated image
— for a slide deck, a blog post, a placeholder asset, concept art, or just for
fun — openimage is for you.

---

## What it does

- Sends a text prompt to a configurable image generation API
- Receives the generated image (base64 or URL, depending on the provider)
- Saves it to a timestamped file (or a named path you specify)
- Renders it inline in your terminal (iTerm2, Kitty, WezTerm)
- Falls back to [chafa](https://github.com/hpjansson/chafa) or a file path on
  unsupported terminals
- Supports batch generation (`-n`) and per-invocation provider/model overrides
- Stores defaults in a local YAML config file so you do not repeat yourself

---

## How it works

```
  User prompt
      |
      v
  Resolve config  ── openimage.yaml (local directory)
      |
      v
  Select provider ── openai / openrouter / stability / replicate / ideogram / ...
      |
      v
  Build request   ── translates flags to provider-specific JSON or multipart body
      |
      v
  Call API        ── POST to provider endpoint with auth header
      |
      v
  Download image  ── base64 decode or fetch URL, write to disk
      |
      v
  Display         ── iTerm2 / Kitty graphics protocol, or chafa fallback
```

---

## Installation

Requires Go 1.22 or later.

```bash
go install github.com/anomalyco/openimage@latest
```

Build from source:

```bash
git clone https://github.com/anomalyco/openimage.git
cd openimage
go build -o openimage .
```

---

## Quick start

```bash
openimage config set api-key sk-or-...
openimage generate "a watercolor painting of mountains at sunset"
```

---

## Configuration

openimage reads `openimage.yaml` from the current working directory. Every
value in the config file can be overridden by a command-line flag.

### Config file

```yaml
api_key: sk-or-v1-abc123
api_base: ""
provider: openrouter
model: openai/dall-e-3
size: 1024x1024
quality: standard
style: vivid
save_dir: ~/Pictures/openimage
display: true
```

### Config subcommands

```
openimage config set <key> <value>    Set a value
openimage config get <key>            Read a value
openimage config list                 Print all values (key is masked)
openimage config path                 Show config file location
```

### Config keys

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `api_key` | string | — | API key for your provider |
| `api_base` | string | — | Override the default API base URL |
| `provider` | string | `openrouter` | Default provider name |
| `model` | string | `openai/dall-e-3` | Default model ID |
| `size` | string | `1024x1024` | Default image dimensions |
| `quality` | string | `standard` | Quality or speed tier (provider-specific) |
| `style` | string | `vivid` | Style preset (provider-specific) |
| `save_dir` | string | `.` | Default output directory (supports `~`) |
| `display` | bool | `true` | Show images inline in the terminal |

### API key resolution

Keys are resolved in this order of priority:

1. `--api-key` flag
2. `OPENIMAGE_API_KEY` environment variable
3. `OPENROUTER_API_KEY` environment variable
4. `api_key` field in `openimage.yaml`

---

## Usage

```
openimage generate [prompt] [flags]
```

The prompt is required. It can be given as a positional argument or with
`--prompt` / `-p`.

```bash
openimage generate "a robot reading a book in a library"
openimage generate -p "a robot reading a book in a library"
```

### Flags

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--prompt` | `-p` | string | Text prompt |
| `--provider` | | string | API provider name |
| `--model` | `-m` | string | Model ID |
| `--size` | `-s` | string | Image dimensions, e.g. `1024x1024` |
| `--quality` | `-q` | string | Quality or speed tier |
| `--style` | | string | Style preset |
| `--output` | `-o` | string | Output file path |
| `--count` | `-n` | int | Number of images (default 1) |
| `--api-base` | | string | Override API base URL |
| `--api-key` | | string | Override API key |
| `--open` | | bool | Open result with the system viewer |

When `--output` is omitted, files are saved to `save_dir` with a timestamped
name: `openimage-20260123-143052.png`. With `-n`, an index is appended:
`openimage-20260123-143052-1.png`, `openimage-20260123-143052-2.png`, etc.

---

## Providers

openimage ships with nine providers. Each handles its own authentication scheme,
endpoint, request format, and response parsing.

| Provider | API base | Auth header | Documentation |
|----------|----------|-------------|---------------|
| `openrouter` | `https://openrouter.ai/api/v1` | `Authorization: Bearer` | [docs](https://openrouter.ai/docs) |
| `openai` | `https://api.openai.com/v1` | `Authorization: Bearer` | [docs](https://platform.openai.com/docs) |
| `stability` | `https://api.stability.ai` | `Authorization: Bearer` | [docs](https://platform.stability.ai/docs) |
| `replicate` | `https://api.replicate.com/v1` | `Authorization: Bearer` | [docs](https://replicate.com/docs) |
| `ideogram` | `https://api.ideogram.ai/v1` | `Api-Key` | [docs](https://docs.ideogram.ai) |
| `deepai` | `https://api.deepai.org` | `api-key` | [docs](https://deepai.org/docs) |
| `getimg` | `https://api.getimg.ai/v2` | `Authorization: Bearer` | [docs](https://docs.getimg.ai) |
| `clipdrop` | `https://clipdrop-api.co` | `x-api-key` | [docs](https://clipdrop.co/apis) |
| `segmind` | `https://api.segmind.com` | `x-api-key` | [docs](https://docs.segmind.com) |

Any OpenAI-compatible endpoint works through the `openai` provider with
`--api-base`:

```bash
# Together AI
openimage generate "prompt" --provider openai \
  --api-base https://api.together.ai/v1 \
  -m black-forest-labs/FLUX.1-schnell

# Hugging Face
openimage generate "prompt" --provider openai \
  --api-base https://router.huggingface.co/v1 \
  -m black-forest-labs/FLUX.1-dev
```

### Selecting a provider

```bash
openimage generate "prompt" --provider replicate -m black-forest-labs/flux-schnell
openimage generate "prompt" --provider stability -m sd3
openimage generate "prompt" --provider ideogram -m v4
openimage generate "prompt" --provider clipdrop
```

Set a default to avoid the flag every time:

```bash
openimage config set provider replicate
```

---

## Models

Model IDs are provider-specific. A few common ones across providers:

| Model | Provider(s) | Notes |
|-------|-------------|-------|
| `openai/dall-e-3` | openrouter, openai | 1024x1024, 1024x1792, 1792x1024 |
| `openai/dall-e-2` | openrouter, openai | 256x256, 512x512, 1024x1024 |
| `sd3` | stability | Stable Diffusion 3 |
| `black-forest-labs/flux-schnell` | replicate | Fast Flux (4 steps) |
| `black-forest-labs/flux-dev` | replicate | Flux Dev |
| `stability-ai/sdxl` | replicate | SDXL |
| `v4` | ideogram | Latest (best text rendering) |
| `seedream-5-lite` | getimg | Fast and capable |
| `sdxl1.0-newreality-lightning` | segmind | Fast SDXL variant |

For the full catalogs, see each provider's model listing:

- [OpenRouter models](https://openrouter.ai/models?modality=image)
- [Replicate models](https://replicate.com/collections/text-to-image)
- [Segmind models](https://platform.segmind.com/explore)
- [GetImg models](https://docs.getimg.ai/reference/models)

---

## Terminal display

openimage uses native terminal graphics protocols to render images inline. No
external tools are required if you use a supported terminal.

| Terminal | Protocol |
|----------|----------|
| iTerm2 | OSC 1337 inline image |
| Kitty | Kitty graphics protocol |
| WezTerm | iTerm2 protocol |

On unsupported terminals, openimage falls back to
[chafa](https://github.com/hpjansson/chafa) if it is installed (install via
`brew install chafa` or your package manager). Without chafa, only the file
path is printed.

Inline display can be disabled:

```bash
openimage config set display false
```

---

## Environment variables

| Variable | Purpose |
|----------|---------|
| `OPENIMAGE_API_KEY` | API key (checked before config file) |
| `OPENROUTER_API_KEY` | API key fallback (compatible with the OpenRouter CLI) |

---

## Examples

```bash
# Quick generation (OpenRouter default)
openimage generate "a cat wearing a space helmet"

# OpenAI DALL-E 3, landscape, HD
openimage generate "cinematic mountain range at golden hour" \
  --provider openai -m dall-e-3 -q hd -s 1792x1024

# Replicate with Flux Schnell
openimage generate "futuristic city at sunset" \
  --provider replicate -m black-forest-labs/flux-schnell

# Stability AI SD3
openimage generate "cyberpunk samurai in rain" \
  --provider stability -m sd3 -s 1024x1024

# Ideogram v4 — great for text in images
openimage generate "a neon sign that says OPENIMAGE" \
  --provider ideogram -m v4

# Batch generation with system viewer
openimage generate "abstract geometric patterns" -n 6 --open

# Together AI via openai provider
openimage generate "a medieval castle on a hill" \
  --provider openai \
  --api-base https://api.together.ai/v1 \
  -m black-forest-labs/FLUX.1-schnell
```

---

## Development

### Prerequisites

- Go 1.22+

### Setup

```bash
git clone https://github.com/anomalyco/openimage.git
cd openimage
go mod tidy
```

### Run

```bash
go run . generate "a test prompt"
go run . config list
```

### Build

```bash
go build -o openimage .
```

### Lint

```bash
go vet ./...
```

### Project layout

```
cmd/
  root.go        Root command, global flags, config loading
  generate.go    generate subcommand
  config.go      config subcommand
pkg/
  config/        YAML config read/write
  display/       Terminal image rendering (iTerm2, Kitty, chafa)
  provider/      Provider interface and implementations
    provider.go    Interface, factory
    openai.go      OpenAI / OpenRouter / OpenAI-compatible
    stability.go   Stability AI
    replicate.go   Replicate
    ideogram.go    Ideogram
    deepai.go      DeepAI
    getimg.go      GetImg.ai
    clipdrop.go    Clipdrop
    segmind.go     Segmind
    util.go        Shared helpers
main.go           Entry point
```

### Adding a new provider

1. Create `pkg/provider/<name>.go`
2. Implement the `Provider` interface:

```go
type Provider interface {
    GenerateImage(params *GenerateParams) (*GenerateResult, error)
    Name() string
}
```

3. Register it in `New()` in `pkg/provider/provider.go`
4. Add it to the help text in `cmd/config.go` and `cmd/generate.go`

---

## Contributing

Bug reports, feature requests, and pull requests are welcome.

1. Fork the repository
2. Create a branch: `git checkout -b my-feature`
3. Make your changes
4. Run `go vet ./...` to check for issues
5. Commit with a clear message
6. Push and open a pull request

Please keep provider additions self-contained (one file in `pkg/provider/`)
and follow the existing patterns for error handling and configuration. Do not
introduce new dependencies unless strictly necessary.

---

## License

MIT — see [LICENSE.md](LICENSE.md).
