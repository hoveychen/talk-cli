// Package runner manages the kokoro engine subprocess.
//
// On first use it downloads the appropriate engine bundle from GitHub Releases
// and the model files from HuggingFace, then invokes the engine binary as a
// subprocess for each TTS request.
package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hoveychen/talk-cli/internal/assets"
	"github.com/hoveychen/talk-cli/internal/downloader"
)

// Options controls the behaviour of New.
type Options struct {
	// NoProgress suppresses the download progress bar.
	NoProgress bool
	// CacheDir overrides the default cache directory (~/.cache/talk-cli).
	CacheDir string
}

// Runner holds paths to cached assets and knows how to invoke the engine.
type Runner struct {
	engineExe  string // absolute path to engine binary
	useMLX     bool   // true when MLX engine is active (darwin/arm64 + lang=en)
	modelPath  string // empty for MLX (engine downloads its own model)
	voicesPath string // empty for MLX
	configPath string // non-empty for zh ONNX (Bopomofo vocab config)
	lang       string // "en" or "zh"
}

// defaultCacheDir returns the platform cache directory for talk-cli.
func defaultCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "talk-cli"), nil
}

// supportedLangs is the set of languages accepted by New.
var supportedLangs = map[string]bool{
	"en": true, "zh": true,
	"es": true, "fr": true, "hi": true,
	"it": true, "ja": true, "pt": true,
}

// modelLang returns the model variant ("en" or "zh") required for lang.
func modelLang(lang string) string {
	if lang == "zh" {
		return "zh"
	}
	return "en"
}

// engineLangCode maps a language tag to the single-letter code the Kokoro
// ONNX engine expects via --lang.
func engineLangCode(lang string) string {
	switch lang {
	case "es":
		return "e"
	case "fr":
		return "f"
	case "hi":
		return "h"
	case "it":
		return "i"
	case "ja":
		return "j"
	case "pt":
		return "p"
	case "zh":
		return "z"
	default:
		return "a" // en-US
	}
}

// New prepares a Runner for the given language, downloading the engine and
// model to the cache directory if they are not already present.
func New(lang string, opts Options) (*Runner, error) {
	if !supportedLangs[lang] {
		return nil, fmt.Errorf("unsupported language %q (want: en, zh, es, fr, hi, it, ja, pt)", lang)
	}

	cacheDir := opts.CacheDir
	if cacheDir == "" {
		var err error
		cacheDir, err = defaultCacheDir()
		if err != nil {
			return nil, err
		}
	}

	r := &Runner{lang: lang}

	// On darwin/arm64, prefer MLX for English; fall back to ONNX on failure.
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" && lang == "en" {
		r.useMLX = true
		if err := r.ensureEngine(cacheDir, opts.NoProgress); err != nil {
			fmt.Fprintf(os.Stderr, "MLX engine unavailable (%v), falling back to ONNX.\n", err)
			r.useMLX = false
		}
	}

	if !r.useMLX {
		if err := r.ensureEngine(cacheDir, opts.NoProgress); err != nil {
			return nil, fmt.Errorf("engine setup: %w", err)
		}
	}
	if !r.useMLX {
		if err := r.ensureModel(cacheDir, opts.NoProgress); err != nil {
			return nil, fmt.Errorf("model setup: %w", err)
		}
	}
	return r, nil
}

// Speak synthesises text and writes a WAV file to outputPath.
// If outputPath is empty a temp file is created; the caller is responsible
// for deleting it.
// When using MLX, if the engine fails at runtime it automatically falls back
// to the ONNX engine (e.g. Metal library unavailable on some macOS versions).
func (r *Runner) Speak(text, voice string, speed float64, outputPath string) (string, error) {
	if outputPath == "" {
		f, err := os.CreateTemp("", "talk-*.wav")
		if err != nil {
			return "", err
		}
		f.Close()
		outputPath = f.Name()
	}

	args := r.speakArgs(text, voice, speed, outputPath)
	if err := r.run(args...); err != nil {
		if r.useMLX {
			// MLX failed at runtime (e.g. Metal unavailable); fall back to ONNX.
			fmt.Fprintf(os.Stderr, "MLX inference failed, falling back to ONNX.\n")
			os.Remove(outputPath)
			if fallbackErr := r.fallbackToONNX(); fallbackErr != nil {
				return "", fmt.Errorf("MLX failed and ONNX fallback also failed: %w", fallbackErr)
			}
			args = r.speakArgs(text, voice, speed, outputPath)
			if err2 := r.run(args...); err2 != nil {
				os.Remove(outputPath)
				return "", err2
			}
			return outputPath, nil
		}
		os.Remove(outputPath)
		return "", err
	}
	return outputPath, nil
}

// fallbackToONNX switches the runner from MLX to the ONNX engine in-place.
// It downloads/verifies the ONNX engine and model if needed.
func (r *Runner) fallbackToONNX() error {
	cacheDir, err := defaultCacheDir()
	if err != nil {
		return err
	}
	r.useMLX = false
	if err := r.ensureEngine(cacheDir, true); err != nil {
		return fmt.Errorf("ONNX engine: %w", err)
	}
	return r.ensureModel(cacheDir, true)
}

// ── internal ──────────────────────────────────────────────────────────────────

func (r *Runner) speakArgs(text, voice string, speed float64, outputPath string) []string {
	if r.useMLX {
		return []string{
			"speak",
			"--text", text,
			"--voice", voice,
			"--speed", fmt.Sprintf("%.2f", speed),
			"--output", outputPath,
		}
	}

	langCode := engineLangCode(r.lang)

	args := []string{
		"speak",
		"--model", r.modelPath,
		"--voices", r.voicesPath,
		"--text", text,
		"--voice", voice,
		"--speed", fmt.Sprintf("%.2f", speed),
		"--lang", langCode,
		"--output", outputPath,
	}
	if r.configPath != "" {
		args = append(args, "--config", r.configPath)
	}
	return args
}

func (r *Runner) run(args ...string) error {
	cmd := exec.Command(r.engineExe, args...) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ── engine management ─────────────────────────────────────────────────────────

func (r *Runner) ensureEngine(cacheDir string, noProgress bool) error {
	var (
		engineURL  string
		engineDir  string
		engineExe  string
		stampFile  string
	)

	if r.useMLX {
		engineURL = assets.MLXEngineURL()
		engineDir = filepath.Join(cacheDir, "engines", "mlx-"+assets.EngineTag+"-"+runtime.GOOS+"-"+runtime.GOARCH)
		if runtime.GOOS == "windows" {
			engineExe = filepath.Join(engineDir, "kokoro_engine_mlx", "kokoro_engine_mlx.exe")
		} else {
			engineExe = filepath.Join(engineDir, "kokoro_engine_mlx", "kokoro_engine_mlx")
		}
	} else {
		var err error
		engineURL, err = assets.EngineURL(runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return err
		}
		engineDir = filepath.Join(cacheDir, "engines", "onnx-"+assets.EngineTag+"-"+runtime.GOOS+"-"+runtime.GOARCH)
		if runtime.GOOS == "windows" {
			engineExe = filepath.Join(engineDir, "kokoro_engine", "kokoro_engine.exe")
		} else {
			engineExe = filepath.Join(engineDir, "kokoro_engine", "kokoro_engine")
		}
	}
	stampFile = filepath.Join(engineDir, ".version")

	// Check if this version is already extracted.
	if stamp, err := os.ReadFile(stampFile); err == nil {
		if strings.TrimSpace(string(stamp)) == assets.EngineTag {
			r.engineExe = engineExe
			return nil
		}
	}

	// Download and extract fresh.
	fmt.Fprintf(os.Stderr, "Setting up engine (%s) ...\n", assets.EngineTag)
	if err := os.RemoveAll(engineDir); err != nil {
		return err
	}
	if err := os.MkdirAll(engineDir, 0o755); err != nil {
		return err
	}

	archiveName := archiveFilename(engineURL)
	if err := downloader.DownloadAndExtract(engineURL, engineDir, noProgress, archiveName); err != nil {
		return fmt.Errorf("downloading engine: %w", err)
	}

	// Make executable on Unix.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(engineExe, 0o755); err != nil {
			return err
		}
	}

	if err := os.WriteFile(stampFile, []byte(assets.EngineTag), 0o644); err != nil {
		return err
	}

	r.engineExe = engineExe
	return nil
}

// ── model management ──────────────────────────────────────────────────────────

func (r *Runner) ensureModel(cacheDir string, noProgress bool) error {
	ml := modelLang(r.lang)
	modelDir := filepath.Join(cacheDir, "models", ml)
	stampFile := filepath.Join(modelDir, ".version")

	// Check if already downloaded.
	if stamp, err := os.ReadFile(stampFile); err == nil {
		if strings.TrimSpace(string(stamp)) == assets.EngineTag {
			return r.setModelPaths(modelDir)
		}
	}

	fmt.Fprintf(os.Stderr, "Downloading %s model ...\n", ml)
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return err
	}

	for _, pair := range assets.ModelFiles(ml) {
		remoteRel, localName := pair[0], pair[1]
		url := assets.ModelURL(remoteRel)
		dest := filepath.Join(modelDir, localName)
		if err := downloader.Download(url, dest, noProgress, localName); err != nil {
			return err
		}
	}

	if err := os.WriteFile(stampFile, []byte(assets.EngineTag), 0o644); err != nil {
		return err
	}
	return r.setModelPaths(modelDir)
}

func (r *Runner) setModelPaths(modelDir string) error {
	r.modelPath = filepath.Join(modelDir, "model.onnx")
	r.voicesPath = filepath.Join(modelDir, "voices.bin")
	if modelLang(r.lang) == "zh" {
		r.configPath = filepath.Join(modelDir, "config.json")
	}
	return nil
}

// Voices returns the list of available voice names by querying the engine.
// This is only used when the hardcoded list needs to be refreshed; most
// callers should use the internal/voices package instead.
func (r *Runner) Voices() ([]string, error) {
	var args []string
	if r.useMLX {
		args = []string{"voices"}
	} else {
		args = []string{"voices", "--model", r.modelPath, "--voices", r.voicesPath}
		if r.configPath != "" {
			args = append(args, "--config", r.configPath)
		}
	}
	cmd := exec.Command(r.engineExe, args...) //nolint:gosec
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("engine error: %w", err)
	}
	var voices []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &voices); err != nil {
		return nil, fmt.Errorf("parsing voice list: %w", err)
	}
	return voices, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func archiveFilename(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return url
}
