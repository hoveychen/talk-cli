// Package downloader provides HTTP download helpers with an optional
// progress bar written to stderr.
package downloader

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Download fetches url and writes the result to destPath atomically
// (writes to destPath+".tmp" first, then renames on success).
// The parent directory of destPath is created if it does not exist.
//
// desc is a short label shown in the progress bar (e.g. "model.onnx").
// When noProgress is false and stderr is a terminal, a live progress line
// is printed; otherwise a single "Downloading …" line is printed instead.
func Download(url, destPath string, noProgress bool, desc string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	resp, err := http.Get(url) //nolint:gosec,noctx
	if err != nil {
		return fmt.Errorf("downloading %s: %w", desc, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: HTTP %d from %s", desc, resp.StatusCode, url)
	}

	total := resp.ContentLength // -1 if unknown

	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	showBar := !noProgress && isTerminal()

	var src io.Reader
	if showBar {
		src = &progressReader{
			r:     resp.Body,
			total: total,
			desc:  desc,
			start: time.Now(),
		}
	} else {
		fmt.Fprintf(os.Stderr, "Downloading %s ...\n", desc)
		src = resp.Body
	}

	_, copyErr := io.Copy(f, src)
	f.Close()

	if showBar {
		fmt.Fprintf(os.Stderr, "\r%-60s\r", "") // erase progress line
		fmt.Fprintf(os.Stderr, "✓ %s\n", desc)
	}

	if copyErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("writing %s: %w", desc, copyErr)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// DownloadAndExtract downloads url to a temp file and extracts it into destDir.
// The archive format is auto-detected from magic bytes (tar.gz or zip).
func DownloadAndExtract(url, destDir string, noProgress bool, desc string) error {
	tmp, err := os.CreateTemp("", "talk-engine-*.tmp")
	if err != nil {
		return err
	}
	tmp.Close()
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := Download(url, tmpPath, noProgress, desc); err != nil {
		return err
	}
	return ExtractArchive(tmpPath, destDir)
}

// ExtractArchive unpacks a tar.gz or zip archive into destDir.
func ExtractArchive(archivePath, destDir string) error {
	data, err := os.ReadFile(archivePath)
	if err != nil {
		return err
	}
	if len(data) < 4 {
		return fmt.Errorf("archive %s is too small", archivePath)
	}
	if data[0] == 0x1f && data[1] == 0x8b {
		return extractTarGz(data, destDir)
	}
	if data[0] == 0x50 && data[1] == 0x4b {
		return extractZip(data, destDir)
	}
	return fmt.Errorf("unknown archive format (magic: %02x %02x)", data[0], data[1])
}

// ── terminal detection ────────────────────────────────────────────────────────

func isTerminal() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ── progressReader ────────────────────────────────────────────────────────────

type progressReader struct {
	r       io.Reader
	total   int64
	done    int64
	desc    string
	start   time.Time
	lastPct int
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.done += int64(n)

	pct := 0
	if p.total > 0 {
		pct = int(p.done * 100 / p.total)
	}
	if pct != p.lastPct || time.Since(p.start) > 200*time.Millisecond {
		p.lastPct = pct
		p.printProgress()
		p.start = time.Now()
	}
	return n, err
}

func (p *progressReader) printProgress() {
	done := float64(p.done) / 1024 / 1024
	label := shorten(p.desc, 30)
	if p.total > 0 {
		total := float64(p.total) / 1024 / 1024
		pct := int(p.done * 100 / p.total)
		fmt.Fprintf(os.Stderr, "\r%-30s  %5.1f / %.1f MB (%3d%%)",
			label, done, total, pct)
	} else {
		fmt.Fprintf(os.Stderr, "\r%-30s  %.1f MB", label, done)
	}
}

func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-(n-1):]
}

// ── archive extraction ────────────────────────────────────────────────────────

func extractTarGz(data []byte, destDir string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, filepath.FromSlash(hdr.Name))
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid tar path: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.Symlink(hdr.Linkname, target); err != nil && !os.IsExist(err) {
				return err
			}
		}
	}
	return nil
}

func extractZip(data []byte, destDir string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		target := filepath.Join(destDir, filepath.FromSlash(f.Name))
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid zip path: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(dst, rc)
		rc.Close()
		dst.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
