// Package player provides cross-platform WAV audio playback.
//
// macOS  — delegates to the built-in `afplay` command (no extra deps).
// Windows — uses PowerShell's System.Media.SoundPlayer (built into .NET,
//            always available on Windows 7+).
package player

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Play blocks until the WAV file at path has finished playing.
func Play(wavPath string) error {
	switch runtime.GOOS {
	case "darwin":
		return playDarwin(wavPath)
	case "windows":
		return playWindows(wavPath)
	default:
		return fmt.Errorf("audio playback not supported on %s — use --output to save the WAV file", runtime.GOOS)
	}
}

func playDarwin(wavPath string) error {
	cmd := exec.Command("afplay", wavPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("afplay: %w\n%s", err, out)
	}
	return nil
}

func playWindows(wavPath string) error {
	// PowerShell one-liner: load the WAV and play it synchronously.
	script := fmt.Sprintf(
		`(New-Object System.Media.SoundPlayer '%s').PlaySync()`,
		wavPath,
	)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("powershell SoundPlayer: %w\n%s", err, out)
	}
	return nil
}
