# Contributing to talk-cli

Thank you for your interest in contributing! Here's how to get started.

## Development setup

**Prerequisites:** Go 1.22+, Python 3.10–3.12 (for engine builds only)

```bash
git clone https://github.com/hoveychen/talk-cli.git
cd talk-cli
go build ./cmd/talk   # build CLI
```

Running `./talk "Hello"` will auto-download the engine and model on first use.

## Project structure

```
cmd/talk/         CLI entry point (Cobra commands)
internal/
  assets/         Version constants and download URLs
  downloader/     HTTP download + archive extraction
  player/         Cross-platform audio playback
  runner/         Engine subprocess management
  voices/         Voice list and descriptions
engine/           Python TTS engine (PyInstaller-packaged at release time)
scripts/          Build and model preparation scripts
```

## Making changes

1. Fork the repository and create a feature branch
2. Make your changes with clear, focused commits
3. Run `go vet ./...` and `go test ./...` to check for issues
4. Open a pull request using the PR template

## Adding a new voice

Voice metadata lives in [internal/voices/voices.go](internal/voices/voices.go). Add an entry to the appropriate slice (`enVoices` or `zhVoices`) and verify it works with the engine:

```bash
./talk -v <new_voice_name> "Test sentence"
```

## Adding a new language

A new language requires:

1. A compatible Kokoro model + voices file
2. Updates to `internal/assets/versions.go` (download URLs)
3. Updates to `internal/runner/runner.go` (engine invocation)
4. New entries in `internal/voices/voices.go`
5. Language detection logic in `cmd/talk/main.go`

Please open an issue first to discuss scope before starting.

## Reporting bugs

Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md). Include your OS, architecture, and the output of:

```bash
talk --no-progress "test" 2>&1
```

## Pull request guidelines

- Keep PRs focused — one feature or fix per PR
- Update documentation if you change behaviour or flags
- Do not commit model files or engine bundles (they are downloaded at runtime and excluded by `.gitignore`)
