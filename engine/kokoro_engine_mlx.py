#!/usr/bin/env python3
"""
Kokoro TTS engine wrapper for Apple Silicon (MLX backend).

Called by the Go CLI as a subprocess. Speaks text or lists voices.
Requires Apple Silicon (M-series) Mac — MLX does not run on Intel or Windows.

Usage:
  kokoro_engine_mlx speak  --model <hf_id_or_path> --voices <ignored>
                            --text <text> --voice <voice> --speed <float>
                            --lang <ignored> --output <wav_path>
  kokoro_engine_mlx voices --model <hf_id_or_path> --voices <ignored>

Notes:
  --model   HuggingFace model ID or local directory.
            Defaults to "mlx-community/Kokoro-82M-bf16".
            On first use the model is downloaded to the HF hub cache
            (~/.cache/huggingface/hub/). Subsequent runs use the cache.
  --voices  Accepted for interface compatibility but ignored; voice
            embeddings are part of the downloaded model.
  --lang    Accepted for interface compatibility but ignored; the voice
            name encodes the language (af_ = American-English, zf_ = zh, …).
"""

import multiprocessing
import sys
import json
import argparse

DEFAULT_MODEL = "mlx-community/Kokoro-82M-bf16"


def load_tts(model_id_or_path: str):
    from kokoro_mlx import KokoroTTS
    return KokoroTTS.from_pretrained(model_id_or_path)


def cmd_speak(args):
    tts = load_tts(args.model)
    result = tts.generate(args.text, voice=args.voice, speed=args.speed)
    import soundfile as sf
    sf.write(args.output, result.audio, result.sample_rate)


def cmd_voices(args):
    tts = load_tts(args.model)
    names = tts.list_voices()
    print(json.dumps(names))


def main():
    parser = argparse.ArgumentParser(prog="kokoro_engine_mlx")
    sub = parser.add_subparsers(dest="command", required=True)

    # ── speak ─────────────────────────────────────────────────────────────────
    sp = sub.add_parser("speak")
    sp.add_argument("--model",  default=DEFAULT_MODEL,
                    help="HuggingFace model ID or local path (default: mlx-community/Kokoro-82M-bf16)")
    sp.add_argument("--voices", default="", help="Ignored — voices are embedded in the model")
    sp.add_argument("--text",   required=True, help="Text to synthesise")
    sp.add_argument("--voice",  default="af_heart", help="Voice name")
    sp.add_argument("--speed",  type=float, default=1.0, help="Speed multiplier")
    sp.add_argument("--lang",   default="",
                    help="Ignored — language is encoded in the voice name")
    sp.add_argument("--output", required=True, help="Output WAV file path")

    # ── voices ────────────────────────────────────────────────────────────────
    vp = sub.add_parser("voices")
    vp.add_argument("--model",  default=DEFAULT_MODEL,
                    help="HuggingFace model ID or local path")
    vp.add_argument("--voices", default="", help="Ignored")

    args = parser.parse_args()

    try:
        if args.command == "speak":
            cmd_speak(args)
        elif args.command == "voices":
            cmd_voices(args)
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    # Required for PyInstaller frozen executables that use multiprocessing.
    multiprocessing.freeze_support()
    main()
