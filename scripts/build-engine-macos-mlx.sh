#!/usr/bin/env bash
# build-engine-macos-mlx.sh
# Builds the PyInstaller bundle for the Apple Silicon MLX-based Kokoro engine.
#
# The MLX engine is language-agnostic — it downloads its own model from
# HuggingFace hub at runtime. One bundle covers both en and zh.
#
# Only arm64 is supported — MLX requires Apple Silicon.
#
# Output:
#   assets/engine-darwin-arm64-mlx.tar.gz
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENGINE_SCRIPT="${REPO_ROOT}/engine/kokoro_engine_mlx.py"
ENGINE_DIR="${REPO_ROOT}/engine"
ASSETS_DIR="${REPO_ROOT}/assets"

# Verify we are on Apple Silicon
if [[ "$(uname -m)" != "arm64" ]]; then
    echo "ERROR: MLX engine can only be built on Apple Silicon (arm64)." >&2
    exit 1
fi

# kokoro-mlx requires Python >=3.10,<3.13.
find_python() {
    for candidate in /opt/anaconda3/bin/python3 /opt/homebrew/bin/python3.12 \
                     /opt/homebrew/bin/python3.11 /opt/homebrew/bin/python3.10 \
                     python3; do
        if command -v "$candidate" &>/dev/null; then
            local ver
            ver=$("$candidate" -c "import sys; print(sys.version_info[:2])" 2>/dev/null)
            if echo "$ver" | grep -qE "\(3, (10|11|12)\)"; then
                echo "$candidate"
                return 0
            fi
        fi
    done
    echo "ERROR: No compatible Python (3.10–3.12) found. kokoro-mlx requires Python < 3.13." >&2
    exit 1
}

PYTHON=$(find_python)
echo "Using Python: ${PYTHON} ($(${PYTHON} --version))"

# ── Build ─────────────────────────────────────────────────────────────────────

VENV_DIR="${ENGINE_DIR}/.venv-mlx-arm64"
DIST_DIR="${ENGINE_DIR}/dist-arm64-mlx"
OUT_ARCHIVE="${ASSETS_DIR}/engine-darwin-arm64-mlx.tar.gz"

echo ""
echo "── Building MLX engine / arm64 ──────────────────────────────────────────"

"${PYTHON}" -m venv "${VENV_DIR}"
PIP="${VENV_DIR}/bin/pip"
PYINSTALLER="${VENV_DIR}/bin/pyinstaller"

$PIP install --quiet --upgrade pip
$PIP install --quiet pyinstaller
$PIP install --quiet "kokoro-mlx" soundfile

# English G2P (misaki) and pre-download spaCy model so PyInstaller can bundle it.
# Without this, misaki tries to call spacy.cli.download() at runtime via
# sys.executable (the frozen binary) which fails inside a PyInstaller bundle.
$PIP install --quiet "misaki[en]" || true
"${VENV_DIR}/bin/python3" -m spacy download en_core_web_sm --quiet || true

rm -rf "${DIST_DIR}"

$PYINSTALLER \
    --noconfirm \
    --onedir \
    --name kokoro_engine_mlx \
    --collect-all kokoro_mlx \
    --collect-all mlx \
    --collect-data misaki \
    --collect-data language_tags \
    --collect-data espeakng_loader \
    --collect-data phonemizer \
    --collect-data spacy \
    --collect-data srsly \
    --collect-data thinc \
    --collect-all en_core_web_sm \
    --distpath "${DIST_DIR}" \
    --workpath "${ENGINE_DIR}/build-arm64-mlx" \
    --specpath "${ENGINE_DIR}" \
    "${ENGINE_SCRIPT}"

# MLX looks for mlx.metallib relative to libmlx.dylib via NSBundle.
# For a PyInstaller onedir bundle, NSBundle resolves resources from the
# _internal/ directory (where libmlx.dylib lives), so copy mlx.metallib there.
echo "  Fixing metallib search path for PyInstaller bundle..."
cp "${DIST_DIR}/kokoro_engine_mlx/_internal/mlx/lib/mlx.metallib" \
   "${DIST_DIR}/kokoro_engine_mlx/_internal/mlx.metallib"

# Ad-hoc sign the bundle so macOS runtime validation passes on 13+.
echo "  Ad-hoc signing bundle..."
codesign --sign - --force --deep "${DIST_DIR}/kokoro_engine_mlx"

mkdir -p "${ASSETS_DIR}"
echo "  Packaging → ${OUT_ARCHIVE}"
tar -czf "${OUT_ARCHIVE}" -C "${DIST_DIR}" kokoro_engine_mlx
echo "  ✓ $(du -sh "${OUT_ARCHIVE}" | cut -f1)  ${OUT_ARCHIVE}"

rm -rf "${DIST_DIR}" "${ENGINE_DIR}/build-arm64-mlx"

echo ""
echo "✓ MLX engine bundle built: ${OUT_ARCHIVE}"
