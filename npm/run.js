#!/usr/bin/env node
"use strict";

const { execFileSync } = require("child_process");
const path = require("path");
const fs = require("fs");

const binName = process.platform === "win32" ? "talk.exe" : "talk";
const binPath = path.join(__dirname, "bin", binName);

if (!fs.existsSync(binPath)) {
  console.error(
    "talk-tts: binary not found. Run `node npm/install.js` or reinstall:\n" +
      "  npm install -g talk-tts"
  );
  process.exit(1);
}

try {
  execFileSync(binPath, process.argv.slice(2), { stdio: "inherit" });
} catch (err) {
  process.exit(err.status ?? 1);
}
