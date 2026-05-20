param(
    [string]$VoskTag       = "v0.3.45",
    [string]$Msys2UcrtBin  = "C:\msys64\ucrt64\bin",
    [string]$ReleaseZipDir = "",
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

$ScriptDir   = $PSScriptRoot
if (-not $ScriptDir) { $ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path }
if (-not $ScriptDir) { throw "Konnte Script-Verzeichnis nicht ermitteln." }

$RepoRootObj = Resolve-Path (Join-Path $ScriptDir "..")
if (-not $RepoRootObj) { throw "Konnte Repo-Root nicht ermitteln." }
$RepoRoot    = $RepoRootObj.Path
Set-Location $RepoRoot

# --- version / release naming ---
$Version = ""
try {
    $tag = git describe --tags --exact-match 2>$null
    if ($tag) { $Version = $tag.TrimStart('v') }
} catch { }
if (-not $Version) { $Version = "dev" }

$BinName   = "OmniPanel-go.exe"
$ZipName   = "OmniPanel-go-windows-amd64-${Version}.zip"

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
$OutExe     = Join-Path $OutDir $BinName

# Release zip destination: prefer explicit param, fall back to $OutDir
if ($ReleaseZipDir -and (Test-Path $ReleaseZipDir)) {
    $ReleaseZip  = Join-Path $ReleaseZipDir $ZipName
} else {
    $ReleaseZip = Join-Path $OutDir $ZipName
}

$StdOutLog  = Join-Path $OutDir "build.stdout.log"
$StdErrLog  = Join-Path $OutDir "build.stderr.log"

if ($Clean) {
    if (Test-Path $VoskRoot)     { Remove-Item -Recurse -Force $VoskRoot }
    if (Test-Path $TmpDir)       { Remove-Item -Recurse -Force $TmpDir }
    if (Test-Path $OutExe)       { Remove-Item -Force $OutExe }
    if (Test-Path $ReleaseZip)   { Remove-Item -Force $ReleaseZip }
    if (Test-Path $StdOutLog)    { Remove-Item -Force $StdOutLog }
    if (Test-Path $StdErrLog)    { Remove-Item -Force $StdErrLog }
}

New-Item -ItemType Directory -Force -Path $ThirdPartyDir  | Out-Null
New-Item -ItemType Directory -Force -Path $TmpDir         | Out-Null
New-Item -ItemType Directory -Force -Path $IncludeDir     | Out-Null
New-Item -ItemType Directory -Force -Path $LibDir         | Out-Null
New-Item -ItemType Directory -Force -Path $BinDir         | Out-Null
New-Item -ItemType Directory -Force -Path $OutDir          | Out-Null

if (Test-Path $Msys2UcrtBin) {
    $env:Path = "$Msys2UcrtBin;$env:Path"
} else {
    Write-Warning "MSYS2 UCRT Bin nicht gefunden: $Msys2UcrtBin"
}

Write-Host "==> Prüfe Toolchain"
gcc --version
go version

$HasLocalVosk = (Test-Path $HeaderPath) -and (Test-Path $DllPath) -and ((Test-Path $LibAPath) -or (Test-Path $LibPath))

if ($HasLocalVosk) {
    Write-Host "==> Verwende vorhandene Vosk-Dateien in $VoskRoot"
} else {
    $ReleaseApiUrl = "https://api.github.com/repos/alphacep/vosk-api/releases/tags/$VoskTag"
    Write-Host "==> Lade Release-Metadaten: $ReleaseApiUrl"

    $apiHeaders = @{
        "User-Agent" = "PowerShell-Vosk-Build-Script"
        "Accept"     = "application/vnd.github+json"
    }
    if ($env:GITHUB_TOKEN) {
        $apiHeaders["Authorization"] = "Bearer $env:GITHUB_TOKEN"
    }

    $release    = $null
    $AssetUrl   = $null

    try {
        $release = Invoke-RestMethod -Uri $ReleaseApiUrl -Headers $apiHeaders
    } catch {
        $statusCode = $null
        if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
            $statusCode = [int]$_.Exception.Response.StatusCode
        }
        $errText = $_.Exception.Message

        if ($statusCode -eq 403 -or $errText -match "rate limit") {
            Write-Warning "GitHub API Rate-Limit erreicht. Versuche Direkt-Download ohne API."
            if (-not $env:GITHUB_TOKEN) {
                Write-Warning "Tipp: Setze GITHUB_TOKEN fuer hoehere API-Limits."
            }
        } else {
            throw
        }
    }

    if ($release -and $release.assets) {
        $AssetUrl = Select-AssetUrl -Assets $release.assets -Patterns @(
            "vosk-win64-*.zip",
            "*win64*.zip"
        )
    }

    if (-not $AssetUrl) {
        $tagNoV = if ($VoskTag.StartsWith("v")) { $VoskTag.Substring(1) } else { $VoskTag }
        $fallbackNames = @(
            "vosk-win64-$tagNoV.zip",
            "vosk-win64-$VoskTag.zip",
            "vosk-win64.zip"
        )

        foreach ($name in $fallbackNames) {
            $candidate = "https://github.com/alphacep/vosk-api/releases/download/$VoskTag/$name"
            if (Test-UrlExists -Url $candidate) {
                $AssetUrl = $candidate
                break
            }
        }
    }

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
}

if (-not (Test-Path $LibAPath) -and (Test-Path $DllPath)) {
    Write-Host "==> Erzeuge libvosk.dll.a aus libvosk.dll"
    $genDefCmd     = Get-Command gendef -ErrorAction SilentlyContinue
    $dllToolCmd    = Get-Command dlltool -ErrorAction SilentlyContinue
    $objdumpCmd    = Get-Command objdump -ErrorAction SilentlyContinue

    if ($dllToolCmd) {
        $DefPath = Join-Path $TmpDir "libvosk.def"
        if (Test-Path $DefPath) { Remove-Item -Force $DefPath }

        if ($genDefCmd) {
            & $genDefCmd.Path $DllPath | Out-Null
            $DefCandidates = @(
                $DefPath,
                (Join-Path (Split-Path -Parent $DllPath) "libvosk.def"),
                (Join-Path $RepoRoot "libvosk.def")
            )
            $GeneratedDef = $DefCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1

            if ($LASTEXITCODE -ne 0 -or -not $GeneratedDef) {
                throw "Konnte Definition nicht aus libvosk.dll erzeugen (gendef fehlgeschlagen)."
            }

            if ($GeneratedDef -ne $DefPath) {
                Copy-Item -Force $GeneratedDef $DefPath
            }
        } elseif ($objdumpCmd) {
            $objdumpLines = & $objdumpCmd.Path -p $DllPath
            $exports = @()
            foreach ($line in $objdumpLines) {
                if ($line -match '^\s*\[\d+\].+\s(vosk_[A-Za-z0-9_]+)$') {
                    $exports += $matches[1]
                }
            }
            $exports       = $exports | Sort-Object -Unique
            if (-not $exports -or $exports.Count -eq 0) {
                throw "Konnte keine vosk_* Exporte in libvosk.dll finden (objdump)."
            }
            @("LIBRARY libvosk.dll", "EXPORTS") + $exports | Set-Content -LiteralPath $DefPath -Encoding Ascii
        } else {
            & $dllToolCmd.Path -z $DefPath --export-all-symbols $DllPath
            if ($LASTEXITCODE -ne 0 -or -not (Test-Path $DefPath)) {
                throw "Konnte Definition nicht aus libvosk.dll erzeugen (dlltool -z fehlgeschlagen)."
            }
        }

        & $dllToolCmd.Path -d $DefPath -D "libvosk.dll" -l $LibAPath
        if ($LASTEXITCODE -ne 0 -or -not (Test-Path $LibAPath)) {
            throw "Konnte Import-Library libvosk.dll.a nicht erzeugen (dlltool fehlgeschlagen)."
        }
    } elseif (-not (Test-Path $LibPath)) {
        throw "Fehlt: MinGW Import-Library libvosk.dll.a (und gendef/dlltool sind nicht verfuegbar)."
    }
}

if (-not (Test-Path $LibAPath)) {
    Write-Warning "libvosk.dll.a fehlt. Build mit GCC/CGO kann mit .lib fehlschlagen."
}

foreach ($dep in @("libgcc_s_seh-1.dll", "libstdc++-6.dll", "libwinpthread-1.dll")) {
    $srcDep = Get-ChildItem -Path $ExtractRoot -Recurse -File -Filter $dep | Select-Object -First 1
    if ($srcDep) { Copy-Item -Force $srcDep.FullName (Join-Path $BinDir $dep) }
}

$env:CGO_ENABLED = "1"
$env:CC          = "gcc"
$env:CXX         = "g++"
$env:CGO_CFLAGS  = "-I$IncludeDir -Wno-error"
$env:CGO_CPPFLAGS = "-I$IncludeDir -Wno-error"
$env:LIBRARY_PATH = $LibDir
$env:Path         = "$BinDir;$env:Path"

Write-Host "==> Build startet (Native Windows, GOOS=windows)"
Write-Host "    stdout: $StdOutLog"
Write-Host "    stderr: $StdErrLog"

if (Test-Path $StdOutLog) { Remove-Item -Force $StdOutLog }
if (Test-Path $StdErrLog) { Remove-Item -Force $StdErrLog }

$proc = Start-Process -FilePath "go" `
    -ArgumentList "build", "-x", "-v", "-o", $OutExe, "." `
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

# Laufzeit-DLLs neben die EXE kopieren
$runtimeDlls = @(
    (Join-Path $BinDir "libvosk.dll"),
    (Join-Path $BinDir "libstdc++-6.dll"),
    (Join-Path $BinDir "libgcc_s_seh-1.dll"),
    (Join-Path $BinDir "libwinpthread-1.dll")
)

foreach ($dll in $runtimeDlls) {
    if (Test-Path $dll) {
        Copy-Item -Force $dll $OutDir
    }
}

# --- Package release zip: exe + user folder + runtime DLLs ---
Write-Host "==> Packe Release-ZIP: $ReleaseZip"

# Create a staging folder for clean zip contents
$StageDir = Join-Path $TmpDir "release-stage"
if (Test-Path $StageDir) { Remove-Item -Recurse -Force $StageDir }
New-Item -ItemType Directory -Force -Path $StageDir | Out-Null

# Copy binary
Copy-Item -Force $OutExe (Join-Path $StageDir $BinName)

# Copy user folder (recursive)
if (Test-Path (Join-Path $RepoRoot "user")) {
    Copy-Item -Recurse -Force (Join-Path $RepoRoot "user") (Join-Path $StageDir "user")
}

# Copy runtime DLLs into the stage folder
foreach ($dll in $runtimeDlls) {
    if (Test-Path $dll) {
        Copy-Item -Force $dll (Join-Path $StageDir (Split-Path $dll -Leaf))
    }
}

# Also copy config.json if it exists at repo root
if (Test-Path (Join-Path $RepoRoot "config.json")) {
    Copy-Item -Force (Join-Path $RepoRoot "config.json") $StageDir
}

# Compress to zip (Compress-Archive is available in PS 5.1 on windows-latest)
Compress-Archive -Path (Get-ChildItem $StageDir | Select-Object -ExpandProperty FullName) -DestinationPath $ReleaseZip -Force

Write-Host ""
Write-Host "Release-ZIP erstellt: $ReleaseZip"

if ($Run) {
    Write-Host "==> Starte $OutExe"
    & $OutExe
}

function Test-UrlExists {
    param([Parameter(Mandatory = $true)][string]$Url)
    try {
        Invoke-WebRequest -Uri $Url -Method Head -UseBasicParsing | Out-Null
        return $true
    } catch {
        return $false
    }
}
