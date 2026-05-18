param(
    [string]$VoskTag = "v0.3.45",
    [string]$Msys2UcrtBin = "C:\msys64\ucrt64\bin",
    [switch]$Clean,
    [switch]$Run
)

$ErrorActionPreference = "Stop"

function Select-AssetUrl {
    param(
        [Parameter(Mandatory = $true)] $Assets,
        [Parameter(Mandatory = $true)][string[]]$Patterns
    )
    foreach ($p in $Patterns) {
        $hit = $Assets | Where-Object { $_.name -like $p } | Select-Object -First 1
        if ($hit) { return $hit.browser_download_url }
    }
    return $null
}

$ScriptDir = $PSScriptRoot
if (-not $ScriptDir) { $ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path }
if (-not $ScriptDir) { throw "Konnte Script-Verzeichnis nicht ermitteln." }

$RepoRootObj = Resolve-Path (Join-Path $ScriptDir "..")
if (-not $RepoRootObj) { throw "Konnte Repo-Root nicht ermitteln." }
$RepoRoot = $RepoRootObj.Path
Set-Location $RepoRoot

$ThirdPartyDir = Join-Path $RepoRoot "third_party"
$VoskRoot      = Join-Path $ThirdPartyDir "vosk"
$TmpDir        = Join-Path $ThirdPartyDir "tmp"

$ZipPath       = Join-Path $TmpDir "vosk.zip"
$ExtractRoot   = Join-Path $TmpDir "vosk-extract"

$IncludeDir = Join-Path $VoskRoot "include"
$LibDir     = Join-Path $VoskRoot "lib"
$BinDir     = Join-Path $VoskRoot "bin"

$HeaderPath = Join-Path $IncludeDir "vosk_api.h"
$DllPath    = Join-Path $BinDir "libvosk.dll"
$LibPath    = Join-Path $LibDir "libvosk.lib"
$LibAPath   = Join-Path $LibDir "libvosk.dll.a"

$OutDir     = Join-Path $RepoRoot "bin"
$OutExe     = Join-Path $OutDir "OmniPanel-go.exe"
$StdOutLog  = Join-Path $OutDir "build.stdout.log"
$StdErrLog  = Join-Path $OutDir "build.stderr.log"

if ($Clean) {
    if (Test-Path $VoskRoot)   { Remove-Item -Recurse -Force $VoskRoot }
    if (Test-Path $TmpDir)     { Remove-Item -Recurse -Force $TmpDir }
    if (Test-Path $OutExe)     { Remove-Item -Force $OutExe }
    if (Test-Path $StdOutLog)  { Remove-Item -Force $StdOutLog }
    if (Test-Path $StdErrLog)  { Remove-Item -Force $StdErrLog }
}

New-Item -ItemType Directory -Force -Path $ThirdPartyDir | Out-Null
New-Item -ItemType Directory -Force -Path $TmpDir | Out-Null
New-Item -ItemType Directory -Force -Path $IncludeDir | Out-Null
New-Item -ItemType Directory -Force -Path $LibDir | Out-Null
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

if (Test-Path $Msys2UcrtBin) {
    $env:Path = "$Msys2UcrtBin;$env:Path"
} else {
    Write-Warning "MSYS2 UCRT Bin nicht gefunden: $Msys2UcrtBin"
}

Write-Host "==> Prüfe Toolchain"
gcc --version
go version

$ReleaseApiUrl = "https://api.github.com/repos/alphacep/vosk-api/releases/tags/$VoskTag"
Write-Host "==> Lade Release-Metadaten: $ReleaseApiUrl"
$release = Invoke-RestMethod -Uri $ReleaseApiUrl -Headers @{ "User-Agent" = "PowerShell-Vosk-Build-Script" }

if (-not $release.assets) { throw "Keine Assets im Release $VoskTag gefunden." }

$AssetUrl = Select-AssetUrl -Assets $release.assets -Patterns @(
    "vosk-win64-*.zip",
    "*win64*.zip"
)
if (-not $AssetUrl) { throw "Kein passendes win64-Asset im Release $VoskTag gefunden." }

Write-Host "==> Verwende Asset: $AssetUrl"
Invoke-WebRequest -Uri $AssetUrl -OutFile $ZipPath

if (Test-Path $ExtractRoot) { Remove-Item -Recurse -Force $ExtractRoot }
New-Item -ItemType Directory -Force -Path $ExtractRoot | Out-Null

Write-Host "==> Entpacke Asset nach $ExtractRoot"
Expand-Archive -Path $ZipPath -DestinationPath $ExtractRoot -Force

$SrcHeader = Get-ChildItem -Path $ExtractRoot -Recurse -File -Filter "vosk_api.h" | Select-Object -First 1
$SrcDll    = Get-ChildItem -Path $ExtractRoot -Recurse -File |
    Where-Object { $_.Name -in @("libvosk.dll", "vosk.dll") } |
    Select-Object -First 1
$SrcLibA   = Get-ChildItem -Path $ExtractRoot -Recurse -File |
    Where-Object { $_.Name -in @("libvosk.dll.a", "libvosk.a") } |
    Select-Object -First 1
$SrcLib    = Get-ChildItem -Path $ExtractRoot -Recurse -File |
    Where-Object { $_.Name -in @("libvosk.lib", "vosk.lib") } |
    Select-Object -First 1

if (-not $SrcHeader) { throw "Fehlt: vosk_api.h im Asset." }
if (-not $SrcDll)    { throw "Fehlt: Runtime-DLL (libvosk.dll oder vosk.dll) im Asset." }
if (-not $SrcLibA -and -not $SrcLib) {
    throw "Fehlt: Import-Library (libvosk.dll.a/libvosk.a/libvosk.lib/vosk.lib) im Asset."
}

Copy-Item -Force $SrcHeader.FullName $HeaderPath
Copy-Item -Force $SrcDll.FullName    $DllPath
if ($SrcLibA) { Copy-Item -Force $SrcLibA.FullName $LibAPath }
if ($SrcLib)  { Copy-Item -Force $SrcLib.FullName  $LibPath }

foreach ($dep in @("libgcc_s_seh-1.dll", "libstdc++-6.dll", "libwinpthread-1.dll")) {
    $srcDep = Get-ChildItem -Path $ExtractRoot -Recurse -File -Filter $dep | Select-Object -First 1
    if ($srcDep) { Copy-Item -Force $srcDep.FullName (Join-Path $BinDir $dep) }
}
$env:CGO_ENABLED  = "1"
$env:CC           = "gcc"
$env:CXX          = "g++"
$env:CGO_CFLAGS   = "-I$IncludeDir -Wno-error"
$env:CGO_CPPFLAGS = "-I$IncludeDir -Wno-error"
$env:CGO_LDFLAGS  = "-L$LibDir -lvosk"
$env:Path         = "$BinDir;$env:Path"

Write-Host "==> Build startet"
Write-Host "    stdout: $StdOutLog"
Write-Host "    stderr: $StdErrLog"

if (Test-Path $StdOutLog) { Remove-Item -Force $StdOutLog }
if (Test-Path $StdErrLog) { Remove-Item -Force $StdErrLog }

$goArgs = @("build", "-x", "-v", "-o", $OutExe, ".")
$proc = Start-Process -FilePath "go" `
    -ArgumentList $goArgs `
    -WorkingDirectory $RepoRoot `
    -NoNewWindow `
    -Wait `
    -PassThru `
    -RedirectStandardOutput $StdOutLog `
    -RedirectStandardError $StdErrLog

if ($proc.ExitCode -ne 0) {
    Write-Host ""
    Write-Host "==> Build fehlgeschlagen (ExitCode $($proc.ExitCode))"
    if (Test-Path $StdErrLog) {
        Write-Host "==> Letzte 120 Zeilen stderr:"
        Get-Content $StdErrLog -Tail 120
    }
    throw "go build fehlgeschlagen. Siehe $StdOutLog und $StdErrLog"
}

if (!(Test-Path $OutExe)) {
    throw "Build meldete Erfolg, aber Binary fehlt: $OutExe"
}

Write-Host ""
Write-Host "Build erfolgreich."
Write-Host "  Binary: $OutExe"
Write-Host "  Header: $HeaderPath"
Write-Host "  DLL:    $DllPath"
Write-Host "  LibDir: $LibDir"

# Laufzeit-DLLs neben die EXE kopieren
$runtimeDlls = @(
    Join-Path $BinDir "libvosk.dll",
    Join-Path $BinDir "libstdc++-6.dll",
    Join-Path $BinDir "libgcc_s_seh-1.dll",
    Join-Path $BinDir "libwinpthread-1.dll"
)

foreach ($dll in $runtimeDlls) {
    if (Test-Path $dll) {
        Copy-Item -Force $dll $OutDir
    }
}

if ($Run) {
    Write-Host "==> Starte $OutExe"
    & $OutExe
}