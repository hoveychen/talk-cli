#!/usr/bin/env python3
"""
Apply dynamic INT8 quantisation to the Kokoro v1.1-zh FP32 ONNX model.

The pre-built FP32 ONNX (~328 MB) is available from:
  https://github.com/thewh1teagle/kokoro-onnx/releases/tag/model-files-v1.1

The conversion is two-stage:
  1. Upgrade the model opset to 20 (required for ConvInteger support in ORT >=1.24)
  2. Dynamic INT8 quantisation (~2.6x size reduction: 328 MB → 125 MB)

Requirements:
    pip install onnx onnxruntime

Usage:
    python3 convert-zh-model.py --input kokoro-v1.1-zh.onnx --output model.onnx
"""

import argparse
import os
import sys
import tempfile


def quantize(input_path: str, output_path: str):
    try:
        import onnx
        from onnx import version_converter
    except ImportError:
        print("ERROR: 'onnx' package not installed. Run: pip install onnx", file=sys.stderr)
        sys.exit(1)

    try:
        from onnxruntime.quantization import quantize_dynamic, QuantType
    except ImportError:
        print("ERROR: 'onnxruntime' package not installed. Run: pip install onnxruntime", file=sys.stderr)
        sys.exit(1)

    fp32_mb = os.path.getsize(input_path) / 1024 / 1024
    print(f"  FP32 ONNX: {fp32_mb:.0f} MB")

    # Stage 1: Upgrade to opset 20 so ConvInteger uses the right kernel in ORT >=1.24
    print("  Upgrading opset to 20 ...")
    with tempfile.NamedTemporaryFile(suffix=".onnx", delete=False) as tmp:
        op20_path = tmp.name
    try:
        model = onnx.load(input_path)
        model20 = version_converter.convert_version(model, 20)
        onnx.save(model20, op20_path)

        # Stage 2: dynamic INT8 quantisation (~2.6x size reduction)
        print("  Quantising to INT8 ...")
        quantize_dynamic(op20_path, output_path, weight_type=QuantType.QInt8)
    finally:
        if os.path.exists(op20_path):
            os.unlink(op20_path)

    int8_mb = os.path.getsize(output_path) / 1024 / 1024
    print(f"✓ INT8 ONNX saved to {output_path} ({int8_mb:.0f} MB, "
          f"{fp32_mb / int8_mb:.1f}x smaller)")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--input",  required=True, help="Input FP32 .onnx file")
    parser.add_argument("--output", required=True, help="Output INT8 .onnx file")
    args = parser.parse_args()
    quantize(args.input, args.output)


if __name__ == "__main__":
    main()
