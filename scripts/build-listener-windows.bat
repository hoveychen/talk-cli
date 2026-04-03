@echo off
REM Build the Windows listener binary using .NET Framework's csc.exe.
REM Requires: Windows with .NET Framework 4.x (pre-installed on Win10+).

setlocal

set CSC=
for /f "delims=" %%i in ('where csc.exe 2^>nul') do (
    set "CSC=%%i"
    goto :found
)

REM Try the .NET Framework directory directly.
if exist "%WINDIR%\Microsoft.NET\Framework64\v4.0.30319\csc.exe" (
    set "CSC=%WINDIR%\Microsoft.NET\Framework64\v4.0.30319\csc.exe"
    goto :found
)
if exist "%WINDIR%\Microsoft.NET\Framework\v4.0.30319\csc.exe" (
    set "CSC=%WINDIR%\Microsoft.NET\Framework\v4.0.30319\csc.exe"
    goto :found
)

echo Error: csc.exe not found. Install .NET Framework 4.x or Visual Studio.
exit /b 1

:found
echo Building with %CSC%...
if not exist bin mkdir bin

set "REF=%WINDIR%\Microsoft.NET\assembly\GAC_MSIL\System.Speech\v4.0_4.0.0.0__31bf3856ad364e35\System.Speech.dll"
if not exist "%REF%" (
    echo Warning: System.Speech.dll not found in GAC, trying bare reference...
    set "REF=System.Speech.dll"
)

"%CSC%" /nologo /optimize /target:winexe ^
    /r:"%REF%" ^
    /r:System.Windows.Forms.dll ^
    /r:System.Drawing.dll ^
    /out:bin\speak-listen.exe ^
    listen\listen_windows.cs

if %ERRORLEVEL% NEQ 0 (
    echo Build failed.
    exit /b 1
)

echo Done: bin\speak-listen.exe
