# openimage

Generate AI images from your terminal. Supports 9+ providers including OpenAI, OpenRouter, Stability AI, Replicate, Ideogram, DeepAI, GetImg.ai, Clipdrop, Segmind, and any OpenAI-compatible endpoint.

## Installation

```bash
go install github.com/anomalyco/openimage@latest
```

Or build from source:

```bash
git clone https://github.com/anomalyco/openimage.git
cd openimage
go build -o openimage .
```

## Quick start

```bash
# Set your API key
openimage config set api-key sk-or-...

# Generate
openimage generate "a cyberpunk cat in a neon-lit alley"
```

## Providers

| Provider | API Key | Popular Models |
|----------|---------|----------------|
| `openrouter` | [OpenRouter key](https://openrouter.ai/keys) | `openai/dall-e-3`, `stability-ai/sdxl`, any OpenRouter model |
| `openai` | [OpenAI key](https://platform.openai.com/api-keys) | `dall-e-3`, `dall-e-2` |
| `stability` | [Stability key](https://platform.stability.ai) | `sd3`, `core`, `ultra` |
| `replicate` | [Replicate token](https://replicate.com/account/api-tokens) | `black-forest-labs/flux-schnell`, `stability-ai/sdxl` |
| `ideogram` | [Ideogram key](https://ideogram.ai/manage-api) | `v3`, `v4` |
| `deepai` | [DeepAI key](https://deepai.org/dashboard/profile) | `standard`, `hd`, `genius` |
| `getimg` | [GetImg key](https://getimg.ai/app/developer/api-keys) | `seedream-5-lite`, `flux-schnell` |
| `clipdrop` | [Clipdrop key](https://clipdrop.co) | n/a (single model) |
| `segmind` | [Segmind key](https://platform.segmind.com/console/api-keys) | `sdxl1.0-newreality-lightning`, `flux-schnell` |

**Also supported via `--provider openai --api-base <url>`:** Together AI (`api.together.ai/v1`), Hugging Face (`router.huggingface.co/v1`), and any OpenAI-compatible endpoint.

Set your default provider:

```bash
openimage config set provider replicate
openimage config set model black-forest-labs/flux-schnell
```

## Commands

### `generate [prompt]`

```bash
# OpenRouter (default)
openimage generate "oil painting of a sailboat" -m openai/dall-e-3 -s 1792x1024

# Stability AI
openimage generate "futuristic city" --provider stability -m sd3 -s 1024x1024

# Replicate
openimage generate "cat in space" --provider replicate -m black-forest-labs/flux-schnell

# Ideogram
openimage generate "logo design" --provider ideogram -m v4

# GetImg.ai
openimage generate "sunset" --provider getimg -m seedream-5-lite -q 2K

# Together AI (openai-compatible)
openimage generate "a dragon" --provider openai --api-base https://api.together.ai/v1 -m black-forest-labs/FLUX.1-schnell

# Batch and open with system viewer
openimage generate "variations" -n 4 --open
```

| Flag | Short | Description |
|------|-------|-------------|
| `--prompt` | `-p` | Text prompt |
| `--provider` | | `openai`, `openrouter`, `stability`, `replicate`, `ideogram`, `deepai`, `getimg`, `clipdrop`, `segmind` |
| `--model` | `-m` | Model ID |
| `--size` | `-s` | Image size, e.g. `1024x1024`, `1792x1024` |
| `--quality` | `-q` | Quality/speed tier (provider-specific) |
| `--style` | | Style preset (provider-specific) |
| `--output` | `-o` | Output file path |
| `--api-base` | | Custom API endpoint |
| `--api-key` | | API key override |
| `--count` | `-n` | Number of images (default: 1) |
| `--open` | | Open with system viewer |

### `config`

```bash
openimage config set api-key sk-...
openimage config set provider replicate
openimage config set model black-forest-labs/flux-schnell
openimage config set save-dir ~/Pictures/openimage
openimage config list
openimage config path
```

Config keys: `api-key`, `api-base`, `provider`, `model`, `size`, `quality`, `style`, `save-dir`, `display`

## Configuration

Config is stored at `openimage.yaml` in the current directory:

```yaml
api_key: sk-...
api_base: ""
provider: openrouter
model: openai/dall-e-3
size: 1024x1024
quality: standard
style: vivid
save_dir: ~/Pictures/openimage
display: true
```

API key resolution order:

1. `--api-key` flag
2. `OPENIMAGE_API_KEY` env var
3. `OPENROUTER_API_KEY` env var
4. Config file

## Terminal image display

Inline display in:

- **iTerm2** — native inline protocol
- **Kitty** — graphics protocol
- **WezTerm** — uses iTerm2 protocol

Falls back to [chafa](https://github.com/hpjansson/chafa) if available. Disable with `openimage config set display false`.

## Provider details

### Replicate
```
openimage generate "prompt" --provider replicate -m black-forest-labs/flux-schnell
openimage generate "prompt" --provider replicate -m black-forest-labs/flux-dev
openimage generate "prompt" --provider replicate -m stability-ai/sdxl
```
Uses `Prefer: wait` for synchronous generation. Model format: `owner/name` or `owner/name:version`.

### Ideogram
```
openimage generate "prompt" --provider ideogram -m v4
openimage generate "prompt" --provider ideogram -m v3 -q TURBO --style GENERAL
```
`--quality` maps to `rendering_speed` (FLASH, TURBO, DEFAULT, QUALITY). `--style` maps to `style_type` (GENERAL, REALISTIC, DESIGN, FICTION).

### DeepAI
```
openimage generate "prompt" --provider deepai -q hd -s 1024x1024
openimage generate "prompt" --provider deepai -q genius --style photography
```
`--quality` maps to `image_generator_version` (standard, hd, genius). `--style` maps to `genius_preference` (anime, photography, graphic, cinematic).

### GetImg.ai
```
openimage generate "prompt" --provider getimg -m seedream-5-lite -q 2K
openimage generate "prompt" --provider getimg -m flux-schnell
```
`--quality` maps to resolution (1K, 2K, 4K). Popular models: `seedream-5-lite`, `seedream-4.0`, `flux-schnell`.

### Clipdrop
```
openimage generate "prompt" --provider clipdrop
```
Fixed 1024x1024 PNG output. No model selection.

### Segmind
```
openimage generate "prompt" --provider segmind -m sdxl1.0-newreality-lightning
openimage generate "prompt" --provider segmind -m flux-schnell
```
473+ models. Supports sync v1 and async v2 with polling.

### Together AI & Hugging Face (openai-compatible)
```
openimage generate "prompt" --provider openai --api-base https://api.together.ai/v1 -m black-forest-labs/FLUX.1-schnell
openimage generate "prompt" --provider openai --api-base https://router.huggingface.co/v1 -m black-forest-labs/FLUX.1-dev
```
Any OpenAI-compatible endpoint works via `--provider openai --api-base <url>`.
