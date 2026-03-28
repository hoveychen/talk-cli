# build-engine-windows.ps1
# Builds PyInstaller bundles for Windows amd64.
# Run this on a Windows machine (or in a Windows CI runner).
#
# Output:
#   assets/en/engine-windows-amd64.zip
#   assets/zh/engine-windows-amd64.zip

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$EngineScript = Join-Path $RepoRoot "engine\kokoro_engine.py"
$EngineDir = Join-Path $RepoRoot "engine"

function Build-Variant {
    param([string]$Variant, [string]$Arch = "amd64")

    Write-Host ""
    Write-Host "── Building $Variant / $Arch ──────────────────────────"

    $VenvDir  = Join-Path $EngineDir ".venv-$Variant-$Arch"
    $DistDir  = Join-Path $EngineDir "dist-$Variant-$Arch"
    $OutDir   = Join-Path $RepoRoot "assets\$Variant"

    # Create venv
    python -m venv $VenvDir
    $pip         = Join-Path $VenvDir "Scripts\pip.exe"
    $pyinstaller = Join-Path $VenvDir "Scripts\pyinstaller.exe"

    & $pip install --quiet --upgrade pip
    & $pip install --quiet pyinstaller

    if ($Variant -eq "en") {
        & $pip install --quiet "kokoro-onnx>=0.4.0" soundfile numpy
        & $pip install --quiet "misaki[en]"
    } else {
        & $pip install --quiet "kokoro-onnx>=0.4.0" soundfile numpy
        & $pip install --quiet "misaki[zh]"
    }

    # PyInstaller build
    if (Test-Path $DistDir) { Remove-Item -Recurse -Force $DistDir }
    & $pyinstaller `
        --noconfirm `
        --onedir `
        --name kokoro_engine `
        --collect-data kokoro_onnx `
        --collect-data misaki `
        --collect-data language_tags `
        --collect-data espeakng_loader `
        --collect-data phonemizer `
        --collect-data spacy `
        --collect-data srsly `
        --collect-data thinc `
        --distpath $DistDir `
        --workpath (Join-Path $EngineDir "build-$Variant-$Arch") `
        --specpath $EngineDir `
        $EngineScript

    # Package as zip
    New-Item -ItemType Directory -Force $OutDir | Out-Null
    $Archive = Join-Path $OutDir "engine-windows-$Arch.zip"
    Write-Host "  Packaging -> $Archive"
    Compress-Archive -Path (Join-Path $DistDir "kokoro_engine") -DestinationPath $Archive -Force
    $Size = [math]::Round((Get-Item $Archive).Length / 1MB, 1)
    Write-Host "  ✓ ${Size}MB  $Archive"

    # Cleanup
    Remove-Item -Recurse -Force $DistDir
    Remove-Item -Recurse -Force (Join-Path $EngineDir "build-$Variant-$Arch") -ErrorAction SilentlyContinue
}

Build-Variant -Variant "en"
Build-Variant -Variant "zh"

Write-Host ""
Write-Host "✓ All Windows engine bundles built."
