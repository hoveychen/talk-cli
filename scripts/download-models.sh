#!/usr/bin/env bash
# download-models.sh  <en|zh>
# Downloads the ONNX model and voices file for the requested variant
# and places them in assets/<variant>/.
set -euo pipefail

VARIANT="${1:?Usage: $0 <en|zh>}"

HF_BASE="https://huggingface.co"
GH_RELEASES="https://github.com/thewh1teagle/kokoro-onnx/releases/download"
ASSETS_DIR="$(cd "$(dirname "$0")/.." && pwd)/assets/${VARIANT}"

# ── helpers ──────────────────────────────────────────────────────────────────

download() {
    local url="$1" dest="$2"
    echo "  Downloading $(basename "$dest") ..."
    if command -v curl &>/dev/null; then
        curl -fL --progress-bar -o "$dest" "$url"
    elif command -v wget &>/dev/null; then
        wget -q --show-progress -O "$dest" "$url"
    else
        echo "ERROR: curl or wget required" >&2; exit 1
    fi
}

# ── English (v1.0 ONNX, int8 ≈ 88 MB; voices from thewh1teagle releases) ─────

download_en() {
    # Model: int8-quantised ONNX from thewh1teagle's GitHub release
    download "${GH_RELEASES}/model-files-v1.0/kokoro-v1.0.int8.onnx" \
             "${ASSETS_DIR}/model.onnx"
    # Voices: packed numpy archive from the same release
    download "${GH_RELEASES}/model-files-v1.0/voices-v1.0.bin" \
             "${ASSETS_DIR}/voices.bin"
}

# ── Chinese (v1.1-zh — download pre-built FP32 ONNX, quantise to INT8) ────────

download_zh() {
    # Pre-built FP32 ONNX (~328 MB) from thewh1teagle/kokoro-onnx releases.
    # We quantise it to INT8 (~82 MB) locally using onnxruntime.

    local GH_V1_1="${GH_RELEASES}/model-files-v1.1"
    local tmp_dir; tmp_dir="$(mktemp -d)"

    # Download pre-built FP32 ONNX
    download "${GH_V1_1}/kokoro-v1.1-zh.onnx" "${tmp_dir}/kokoro-v1.1-zh.onnx"

    # Install quantisation dependencies if not already present
    echo "  Installing onnx + onnxruntime for INT8 quantisation..."
    pip install --quiet onnx onnxruntime

    # Quantise FP32 → INT8 (~4x size reduction)
    echo "  Quantising FP32 → INT8 ..."
    python3 "$(dirname "$0")/convert-zh-model.py" \
        --input  "${tmp_dir}/kokoro-v1.1-zh.onnx" \
        --output "${ASSETS_DIR}/model.onnx"

    # Download pre-packed voices.bin for v1.1-zh
    download "${GH_V1_1}/voices-v1.1-zh.bin" "${ASSETS_DIR}/voices.bin"

    rm -rf "${tmp_dir}"
}

# ── Main ─────────────────────────────────────────────────────────────────────

mkdir -p "${ASSETS_DIR}"

case "${VARIANT}" in
    en) download_en ;;
    zh) download_zh ;;
    *)  echo "ERROR: unknown variant '${VARIANT}', expected en or zh" >&2; exit 1 ;;
esac

echo "✓ ${VARIANT} assets saved to ${ASSETS_DIR}/"
