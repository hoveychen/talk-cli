#!/usr/bin/env node
"use strict";

const https = require("https");
const fs = require("fs");
const path = require("path");

// Platform → GitHub Release asset mapping
const PLATFORMS = {
  "darwin-arm64": "speak-darwin-arm64",
  "darwin-x64": "speak-darwin-amd64",
  "win32-x64": "speak-windows-amd64.exe",
};

const pkg = require("../package.json");
const version = pkg.version;
const key = `${process.platform}-${process.arch}`;
const asset = PLATFORMS[key];

if (!asset) {
  console.error(
    `speak-cli: unsupported platform ${process.platform}/${process.arch}\n` +
      `Supported: macOS (arm64, x64), Windows (x64)\n` +
      `Build from source: https://github.com/hoveychen/speak-cli`
  );
  process.exit(1);
}

const binDir = path.join(__dirname, "bin");
const binName = process.platform === "win32" ? "speak.exe" : "speak";
const binPath = path.join(binDir, binName);
const url = `https://github.com/hoveychen/speak-cli/releases/download/v${version}/${asset}`;

function download(url, dest, redirects) {
  if (redirects <= 0) {
    console.error("speak-cli: too many redirects");
    process.exit(1);
  }

  return new Promise((resolve, reject) => {
    const mod = url.startsWith("https") ? https : require("http");
    mod
      .get(url, (res) => {
        // Follow redirects (GitHub uses 302)
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          return resolve(download(res.headers.location, dest, redirects - 1));
        }

        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`HTTP ${res.statusCode} downloading ${url}`));
        }

        const totalBytes = parseInt(res.headers["content-length"], 10) || 0;
        let downloaded = 0;

        const file = fs.createWriteStream(dest);
        res.on("data", (chunk) => {
          downloaded += chunk.length;
          if (totalBytes > 0 && process.stderr.isTTY) {
            const pct = ((downloaded / totalBytes) * 100).toFixed(0);
            process.stderr.write(`\rspeak-cli: downloading ${pct}%`);
          }
        });
        res.pipe(file);
        file.on("finish", () => {
          if (process.stderr.isTTY) process.stderr.write("\n");
          file.close(resolve);
        });
        file.on("error", reject);
      })
      .on("error", reject);
  });
}

async function main() {
  fs.mkdirSync(binDir, { recursive: true });

  console.log(`speak-cli: downloading speak v${version} for ${key}...`);
  await download(url, binPath, 5);

  if (process.platform !== "win32") {
    fs.chmodSync(binPath, 0o755);
  }

  console.log(`speak-cli: installed to ${binPath}`);
}

main().catch((err) => {
  console.error(`speak-cli: ${err.message}`);
  process.exit(1);
});
