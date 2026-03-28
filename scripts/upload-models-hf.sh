#!/usr/bin/env bash
# upload-models-hf.sh
#
# Uploads the prepared ONNX model and voices.bin for each variant (en, zh)
# to a HuggingFace repository.
#
# Prerequisites:
#   1. Log in:   hf auth login
#   2. Prepare:  make prepare-en && make prepare-zh  (or just make prepare)
#
# Usage:
#   bash scripts/upload-models-hf.sh --repo <owner/repo-name> [--variant en|zh|all]
#
# Example:
#   bash scripts/upload-models-hf.sh --repo hoveychen/kokoro-models
#   bash scripts/upload-models-hf.sh --repo hoveychen/kokoro-models --variant en
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HF_CMD="$(command -v hf 2>/dev/null || echo /opt/anaconda3/bin/hf)"

# ── argument parsing ──────────────────────────────────────────────────────────

REPO=""
VARIANT="all"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --repo)    REPO="$2";    shift 2 ;;
        --variant) VARIANT="$2"; shift 2 ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

if [[ -z "$REPO" ]]; then
    echo "Usage: $0 --repo <owner/repo-name> [--variant en|zh|all]" >&2
    exit 1
fi

# ── helpers ───────────────────────────────────────────────────────────────────

upload_variant() {
    local variant="$1"
    local assets_dir="${REPO_ROOT}/assets/${variant}"
    local model_path="${assets_dir}/model.onnx"
    local voices_path="${assets_dir}/voices.bin"

    echo ""
    echo "── Uploading ${variant} assets → ${REPO} ─────────────────────"

    # Validate files exist and are real (> 1 MB)
    if [[ ! -f "$model_path" || $(wc -c < "$model_path") -lt 1048576 ]]; then
        echo "  ERROR: ${model_path} is missing or a placeholder." >&2
        echo "         Run: make prepare-${variant}" >&2
        return 1
    fi
    if [[ ! -f "$voices_path" || $(wc -c < "$voices_path") -lt 1024 ]]; then
        echo "  ERROR: ${voices_path} is missing or a placeholder." >&2
        echo "         Run: make prepare-${variant}" >&2
        return 1
    fi

    # Create the repository if it doesn't exist yet
    echo "  Ensuring repository ${REPO} exists..."
    "$HF_CMD" repo create "${REPO}" --type model 2>/dev/null || true

    echo "  Uploading model.onnx ($(du -sh "$model_path" | cut -f1))..."
    "$HF_CMD" upload "${REPO}" "${model_path}" "${variant}/model.onnx"

    echo "  Uploading voices.bin ($(du -sh "$voices_path" | cut -f1))..."
    "$HF_CMD" upload "${REPO}" "${voices_path}" "${variant}/voices.bin"

    # Upload config.json for zh (required for Bopomofo tokenisation in v1.1-zh)
    if [[ "$variant" == "zh" ]]; then
        local config_path="${assets_dir}/config.json"
        if [[ -f "$config_path" ]]; then
            echo "  Uploading config.json..."
            "$HF_CMD" upload "${REPO}" "${config_path}" "zh/config.json"
        else
            echo "  WARNING: assets/zh/config.json not found — skipping." >&2
        fi
    fi

    echo "  ✓ ${variant} assets uploaded."
}

# ── check HF login ────────────────────────────────────────────────────────────

echo "Checking HuggingFace login..."
if ! "$HF_CMD" auth whoami &>/dev/null; then
    echo "ERROR: Not logged in to HuggingFace." >&2
    echo "       Run: hf auth login" >&2
    exit 1
fi

USERNAME=$("$HF_CMD" auth whoami 2>/dev/null | head -1 || echo "unknown")
echo "Logged in as: ${USERNAME}"

# ── main ──────────────────────────────────────────────────────────────────────

case "${VARIANT}" in
    en)  upload_variant en ;;
    zh)  upload_variant zh ;;
    all) upload_variant en; upload_variant zh ;;
    *)
        echo "ERROR: unknown variant '${VARIANT}', expected en, zh, or all" >&2
        exit 1
        ;;
esac

echo ""
echo "✓ Upload complete."
echo "  View at: https://huggingface.co/${REPO}"
