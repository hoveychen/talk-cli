---
name: talk-tts
description: Multilingual text-to-speech CLI — synthesize speech, select voices, export audio
---

# talk-tts

`talk` is a fast, offline-capable multilingual text-to-speech CLI powered by Kokoro. It auto-detects language, supports 150+ voices across 8 languages, and uses the faster MLX backend automatically on Apple Silicon.

After first use, the engine and models are cached locally — subsequent runs start instantly with no network required.

## Installation

Before using talk commands, verify it is installed:

```bash
talk --help
```

If not installed, use one of:

```bash
# npm (Node.js 18+)
npm install -g talk-tts

# macOS
curl -fsSL https://raw.githubusercontent.com/hoveychen/talk-cli/main/install.sh | bash

# Windows PowerShell
powershell -c "irm https://raw.githubusercontent.com/hoveychen/talk-cli/main/install.ps1 | iex"
```

## Commands Reference

### `talk [flags] <text>` — Synthesize speech

Speak text aloud or save to WAV. Language is auto-detected from the text unless `--lang` is specified.

| Flag | Default | Description |
|------|---------|-------------|
| `--lang` | `auto` | Language: `auto`, `en`, `zh`, `es`, `fr`, `hi`, `it`, `ja`, `pt` |
| `-v, --voice` | per-language default | Voice name (see voice naming below) |
| `-s, --speed` | `1.0` | Speed multiplier, range `0.5`–`2.0` |
| `-o, --output` | _(play audio)_ | Save WAV to file instead of playing |
| `--no-progress` | `false` | Suppress download progress bar |

**Important:** The text argument must be a single quoted string. Wrap it in quotes.

### `talk voices [--lang <lang>]` — List available voices

Lists all available voices offline (no engine or model download needed).

- `--lang all` (default): show all voices
- `--lang en`: English voices only
- `--lang zh`: Chinese voices only
- Also supports: `es`, `fr`, `hi`, `it`, `ja`, `pt`

### `talk init [--lang <lang>]` — Pre-download assets

Downloads engine and model files for offline use.

- `--lang all` (default): download all languages
- `--lang en`: English only (~88 MB model + engine)
- `--lang zh`: Chinese only (~82 MB model + engine)

## Language Detection

When `--lang` is `auto` (default), talk inspects the text for CJK characters (Unicode U+4E00–U+9FFF):
- If any CJK character is found → Chinese (`zh`)
- Otherwise → English (`en`)

For other languages (Spanish, French, Hindi, Italian, Japanese, Portuguese), you **must** specify `--lang` explicitly.

**Supported languages:** `en`, `zh`, `es`, `fr`, `hi`, `it`, `ja`, `pt`

## Voice Naming Convention

Voices follow the pattern `{language}{gender}_{name}`:

| Prefix | Language | Gender |
|--------|----------|--------|
| `af_` | American English | Female |
| `am_` | American English | Male |
| `bf_` | British English | Female |
| `bm_` | British English | Male |
| `ef_` | Spanish | Female |
| `em_` | Spanish | Male |
| `ff_` | French | Female |
| `hf_` | Hindi | Female |
| `hm_` | Hindi | Male |
| `if_` | Italian | Female |
| `im_` | Italian | Male |
| `jf_` | Japanese | Female |
| `jm_` | Japanese | Male |
| `pf_` | Portuguese (BR) | Female |
| `pm_` | Portuguese (BR) | Male |
| `zf_` | Mandarin Chinese | Female |
| `zm_` | Mandarin Chinese | Male |

**Default voices per language:**

| Language | Default Voice |
|----------|--------------|
| English | `af_heart` |
| Chinese | `zf_001` |
| Spanish | `ef_dora` |
| French | `ff_siwis` |
| Hindi | `hf_alpha` |
| Italian | `if_sara` |
| Japanese | `jf_alpha` |
| Portuguese | `pf_dora` |

Use `talk voices --lang <code>` to see all available voices for a language.

## Common Patterns

```bash
# Basic speech (language auto-detected)
talk "Hello, world!"
talk "你好，欢迎使用 talk-cli"

# Choose a specific voice
talk -v af_sky "A different voice"
talk -v zm_010 "换一个男声"

# Adjust speed
talk -s 1.5 "Speak faster"
talk -s 0.7 "Speak slower"

# Save to WAV file
talk -o greeting.wav "Hello, world!"

# Force language (required for es/fr/hi/it/ja/pt)
talk --lang ja "こんにちは"
talk --lang es "Hola, mundo"

# Scripting: suppress progress bar
talk --no-progress -o out.wav "Silent download"

# Batch export
for i in 1 2 3; do
  talk -o "part${i}.wav" "This is part ${i}"
done
```

## Offline Setup

For offline environments, pre-download assets:

```bash
talk init              # All languages
talk init --lang en    # English only
talk init --lang zh    # Chinese only
```

Assets are cached in `~/.cache/talk-cli/`:
- Engine: PyInstaller-packaged Kokoro engine (~200 MB)
- Models: ONNX model + voice data (~80–90 MB per language)
- On Apple Silicon: MLX engine is also downloaded for English (faster)

## Platform Support

| Platform | Architecture | Backend | Notes |
|----------|-------------|---------|-------|
| macOS | Apple Silicon (arm64) | MLX + ONNX fallback | Fastest on M-series |
| macOS | Intel (amd64) | ONNX | |
| Windows | amd64 | ONNX | |
| Linux | — | Export only | Use `--output` to save WAV; no audio playback |

## Troubleshooting

- **First run is slow**: The engine and model are downloaded on first use. Use `talk init` to pre-download.
- **MLX fallback message on Apple Silicon**: If the MLX engine cannot load (e.g., missing Metal libraries), talk automatically falls back to the ONNX engine. This is normal and does not affect output quality.
- **"unsupported platform" error**: talk currently supports macOS and Windows. On Linux, build from source.
- **Output format**: Only WAV output is supported. Use external tools (ffmpeg) to convert to other formats.

## Constraints

- Text must be passed as a **single argument** — always wrap in quotes
- Speed range: `0.5` to `2.0`
- Output format: WAV only (use `--output` flag)
- No streaming or pipe support — the full audio is synthesized before playback
- Auto-detection only distinguishes English and Chinese; other languages require `--lang`
