#!/usr/bin/env bash
# build-engine-macos.sh
# Builds a universal ONNX engine bundle for macOS (handles both en and zh).
#
# Output:
#   assets/engine-darwin-arm64-onnx.tar.gz
#   assets/engine-darwin-amd64-onnx.tar.gz  (if x86_64 Python is available)
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENGINE_SCRIPT="${REPO_ROOT}/engine/kokoro_engine.py"
ENGINE_DIR="${REPO_ROOT}/engine"
ASSETS_DIR="${REPO_ROOT}/assets"

HOST_ARCH="$(uname -m)"

# ── helpers ──────────────────────────────────────────────────────────────────

# misaki[zh]>=0.9.0 requires Python >=3.8,<3.13.
# Return the first compatible Python found (prefer 3.12, 3.11, 3.10).
find_python_compat() {
    for candidate in \
        /opt/homebrew/bin/python3.12 \
        /opt/homebrew/bin/python3.11 \
        /opt/homebrew/bin/python3.10 \
        /opt/anaconda3/bin/python3.12 \
        /opt/anaconda3/bin/python3.11 \
        /opt/anaconda3/bin/python3.10 \
        /usr/local/bin/python3.12 \
        /usr/local/bin/python3.11 \
        /usr/local/bin/python3.10; do
        if command -v "$candidate" &>/dev/null; then
            local ver
            ver=$("$candidate" -c "import sys; print(sys.version_info[:2])" 2>/dev/null)
            if echo "$ver" | grep -qE "\(3, (10|11|12)\)"; then
                echo "$candidate"
                return 0
            fi
        fi
    done
    # Fall back to default python3 if it's in range
    if python3 -c "import sys; assert (3,8) <= sys.version_info < (3,13)" 2>/dev/null; then
        echo "python3"
        return 0
    fi
    echo "ERROR: No compatible Python (3.10–3.12) found for building ONNX engine." >&2
    echo "       Install via: brew install python@3.12" >&2
    exit 1
}

PYTHON_COMPAT="$(find_python_compat)"
echo "Using Python: ${PYTHON_COMPAT} ($(${PYTHON_COMPAT} --version))"

can_build_arch() {
    local arch="$1"
    if [[ "$arch" == "amd64" ]]; then
        arch -x86_64 "${PYTHON_COMPAT}" --version &>/dev/null 2>&1
    else
        "${PYTHON_COMPAT}" --version &>/dev/null 2>&1
    fi
}

# build_onnx <arm64|amd64>
# Builds the universal ONNX engine bundle for the given arch.
# Installs both en and zh G2P deps so the single bundle handles all languages.
build_onnx() {
    local arch="$1"
    local venv_dir="${ENGINE_DIR}/.venv-onnx-${arch}"
    local dist_dir="${ENGINE_DIR}/dist-${arch}-onnx"
    local out_archive="${ASSETS_DIR}/engine-darwin-${arch}-onnx.tar.gz"

    echo ""
    echo "── Building universal ONNX engine / ${arch} ──────────────────────────"

    if ! can_build_arch "$arch"; then
        echo "  ⚠️  No $([ "$arch" = "amd64" ] && echo "x86_64" || echo "arm64") Python found — skipping."
        return 0
    fi

    # Create venv
    if [[ "$arch" == "amd64" ]]; then
        arch -x86_64 "${PYTHON_COMPAT}" -m venv "${venv_dir}"
        local python="${venv_dir}/bin/python3"
        local pip="${venv_dir}/bin/pip"
        local pyinstaller="arch -x86_64 ${venv_dir}/bin/pyinstaller"
    else
        "${PYTHON_COMPAT}" -m venv "${venv_dir}"
        local python="${venv_dir}/bin/python3"
        local pip="${venv_dir}/bin/pip"
        local pyinstaller="${venv_dir}/bin/pyinstaller"
    fi

    $pip install --quiet --upgrade pip
    $pip install --quiet pyinstaller

    # Core TTS dependencies
    $pip install --quiet "kokoro-onnx>=0.4.0" soundfile numpy

    # English phonemization: kokoro_onnx uses phonemizer + espeakng_loader internally.
    # misaki[en] / spacy are NOT needed for ONNX (only for MLX).

    # Chinese G2P: misaki[zh]>=0.9.0 for ZHG2P(version='1.1') + jieba + pypinyin_dict
    $pip install --quiet "misaki[zh]>=0.9.0" ordered-set pypinyin_dict || true

    # Build
    rm -rf "${dist_dir}"

    # shellcheck disable=SC2086
    $pyinstaller \
        --noconfirm \
        --onedir \
        --name kokoro_engine \
        --collect-data kokoro_onnx \
        --collect-data misaki \
        --collect-all jieba \
        --collect-data ordered_set \
        --collect-all pypinyin_dict \
        --collect-data language_tags \
        --collect-data espeakng_loader \
        --collect-data phonemizer \
        --distpath "${dist_dir}" \
        --workpath "${ENGINE_DIR}/build-${arch}-onnx" \
        --specpath "${ENGINE_DIR}" \
        "${ENGINE_SCRIPT}"

    # Ad-hoc sign the bundle so macOS runtime validation passes on 13+.
    # This does not require an Apple Developer account.
    echo "  Ad-hoc signing bundle..."
    codesign --sign - --force --deep "${dist_dir}/kokoro_engine"

    mkdir -p "${ASSETS_DIR}"
    echo "  Packaging → ${out_archive}"
    tar -czf "${out_archive}" -C "${dist_dir}" kokoro_engine
    echo "  ✓ $(du -sh "${out_archive}" | cut -f1)  ${out_archive}"

    rm -rf "${dist_dir}" "${ENGINE_DIR}/build-${arch}-onnx"
}

# ── Main ─────────────────────────────────────────────────────────────────────

# Accept optional arch argument: arm64, amd64, or all (default).
REQUESTED_ARCH="${1:-all}"

if [[ "$REQUESTED_ARCH" == "all" || "$REQUESTED_ARCH" == "arm64" ]]; then
    build_onnx arm64
fi
if [[ "$REQUESTED_ARCH" == "all" || "$REQUESTED_ARCH" == "amd64" ]]; then
    build_onnx amd64
fi

echo ""
echo "Done: macOS ONNX engine bundles built."
