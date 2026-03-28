#!/usr/bin/env python3
"""
Kokoro TTS engine wrapper.

Called by the Go CLI as a subprocess. Speaks text or lists voices.

Usage:
  kokoro_engine speak  --model <path> --voices <path> --text <text>
                       --voice <voice> --speed <float> --lang <code>
                       --output <wav_path>
  kokoro_engine voices --model <path> --voices <path>
"""

import sys
import json
import argparse
import os


def load_kokoro(model_path: str, voices_path: str, config_path: str = None):
    from kokoro_onnx import Kokoro
    if config_path:
        return Kokoro(model_path, voices_path, vocab_config=config_path)
    return Kokoro(model_path, voices_path)


def phonemize_zh(text: str) -> str:
    """Convert Chinese text to phonemes using misaki ZHG2P(version='1.1').

    version='1.1' uses ZHFrontend (Zhuyin-based) which matches the training
    phoneme format of the kokoro-v1.1-zh model and produces correct quality.
    Compatible with both misaki 0.7.x (returns str) and 0.9.x+ (returns (str, None)).
    """
    from misaki import zh as mzh
    g2p = mzh.ZHG2P(version='1.1')
    result = g2p(text)
    # misaki >=0.9: returns (phonemes, extra); misaki 0.7: returns plain str
    if isinstance(result, tuple):
        return result[0]
    return result


def cmd_speak(args):
    config = getattr(args, 'config', None) or None
    kokoro = load_kokoro(args.model, args.voices, config)
    if args.lang == "z":
        # Chinese: misaki ZHG2P(version='1.1') → Bopomofo phonemes → kokoro
        phonemes = phonemize_zh(args.text)
        samples, sample_rate = kokoro.create(
            phonemes,
            voice=args.voice,
            speed=args.speed,
            is_phonemes=True,
        )
    else:
        # kokoro_onnx v0.5.0+ expects BCP-47 lang codes for phonemization.
        _LANG_MAP = {
            "a": "en-us",
            "b": "en-gb",
            "j": "ja",
            "z": "zh",
            "e": "es",
            "f": "fr-fr",
            "h": "hi",
            "i": "it",
            "p": "pt-br",
        }
        bcp47 = _LANG_MAP.get(args.lang, args.lang)
        samples, sample_rate = kokoro.create(
            args.text,
            voice=args.voice,
            speed=args.speed,
            lang=bcp47,
            is_phonemes=False,
        )
    import soundfile as sf
    sf.write(args.output, samples, sample_rate)


def cmd_voices(args):
    # voices.bin is a numpy archive: keys are voice names, values are embeddings.
    import numpy as np
    try:
        data = np.load(args.voices, allow_pickle=True).item()
        names = sorted(data.keys())
    except Exception:
        # Fallback: instantiate Kokoro and call get_voices() if available.
        config = getattr(args, 'config', None) or None
        kokoro = load_kokoro(args.model, args.voices, config)
        if hasattr(kokoro, "get_voices"):
            names = sorted(kokoro.get_voices())
        else:
            print("[]")
            return
    print(json.dumps(names))


def main():
    parser = argparse.ArgumentParser(prog="kokoro_engine")
    sub = parser.add_subparsers(dest="command", required=True)

    # ── speak ─────────────────────────────────────────────────────────────────
    sp = sub.add_parser("speak")
    sp.add_argument("--model",  required=True, help="Path to model.onnx")
    sp.add_argument("--voices", required=True, help="Path to voices.bin")
    sp.add_argument("--text",   required=True, help="Text to synthesise")
    sp.add_argument("--voice",  default="af_sky", help="Voice name")
    sp.add_argument("--speed",  type=float, default=1.0, help="Speed multiplier")
    sp.add_argument("--lang",   default="a",
                    help="Language code: a=en-US, b=en-GB, j=ja, z=zh, "
                         "e=es, f=fr, h=hi, i=it, p=pt-BR")
    sp.add_argument("--config", default="", help="Path to model config.json for custom vocab (zh)")
    sp.add_argument("--output", required=True, help="Output WAV file path")

    # ── voices ────────────────────────────────────────────────────────────────
    vp = sub.add_parser("voices")
    vp.add_argument("--model",  required=True, help="Path to model.onnx")
    vp.add_argument("--voices", required=True, help="Path to voices.bin")
    vp.add_argument("--config", default="", help="Path to model config.json for custom vocab (zh)")

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
    main()
