param(
    [string]$VoskTag = "v0.3.45",
    [string]$Msys2UcrtBin = "C:\msys64\ucrt64\bin",
    [string]$GitHubToken = $env:GITHUB_TOKEN,
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
    if ($GitHubToken) {
        $apiHeaders["Authorization"] = "Bearer $GitHubToken"
    }

    $release = $null
    $AssetUrl = $null

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
            if (-not $GitHubToken) {
                Write-Warning "Tipp: Setze GITHUB_TOKEN fuer hoehere API-Limits (z.B. `$env:GITHUB_TOKEN='ghp_...')."
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
    $genDefCmd = Get-Command gendef -ErrorAction SilentlyContinue
    $dllToolCmd = Get-Command dlltool -ErrorAction SilentlyContinue
    $objdumpCmd = Get-Command objdump -ErrorAction SilentlyContinue

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
            $exports = $exports | Sort-Object -Unique
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
    Write-Warning "libvosk.dll.a fehlt. Build mit GCC/CGO kann mit .lib fehlschlagen. Installiere mindestens dlltool + objdump (MSYS2: mingw-w64-ucrt-x86_64-tools)."
}

foreach ($dep in @("libgcc_s_seh-1.dll", "libstdc++-6.dll", "libwinpthread-1.dll")) {
    $srcDep = Get-ChildItem -Path $ExtractRoot -Recurse -File -Filter $dep | Select-Object -First 1
    if ($srcDep) { Copy-Item -Force $srcDep.FullName (Join-Path $BinDir $dep) }
}
# ---------------------------------------------------------------------------
# Lege vosk_api.h und libvosk.dll.a in das src/-Verzeichnis, das die
# Vosk-Go-Package per '#cgo CPPFLAGS: -I ${SRCDIR}/../src' erwartet.
# Dadurch werden KEINE globalen CGO_CFLAGS/CGO_CPPFLAGS benötigt, die
# sonst auch runtime/cgo treffen und unter '-Wall -Werror' des Go-Toolchains
# zu 'cgo.exe: exit status 2' führen.
# ---------------------------------------------------------------------------
Write-Host "==> Ermittle Vosk-Go-Modulpfad"
$VoskGoModDir = & go list -m -f '{{.Dir}}' github.com/alphacep/vosk-api/go 2>&1
if ($LASTEXITCODE -ne 0 -or -not $VoskGoModDir) {
    throw "Konnte Vosk-Go-Modulpfad nicht ermitteln: $VoskGoModDir"
}
Write-Host "    Vosk-Go-Modul: $VoskGoModDir"

# ${SRCDIR}/../src relativ zum Go-Paket-Verzeichnis
$VoskModParent = Split-Path -Parent $VoskGoModDir
$VoskModSrcDir = Join-Path $VoskModParent "src"

Write-Host "    Ziel src/: $VoskModSrcDir"
New-Item -ItemType Directory -Force -Path $VoskModSrcDir | Out-Null

# Schreibschutz im Modul-Cache aufheben (Go setzt Dateien auf read-only)
Get-ChildItem $VoskModSrcDir -ErrorAction SilentlyContinue | ForEach-Object {
    $_.Attributes = $_.Attributes -band (-bnot [System.IO.FileAttributes]::ReadOnly)
}
# Verzeichnis selbst ebenfalls beschreibbar machen
try { attrib -R "$VoskModSrcDir" /D } catch {}

Copy-Item -Force $HeaderPath (Join-Path $VoskModSrcDir "vosk_api.h")
Write-Host "    vosk_api.h kopiert nach $VoskModSrcDir"

if (Test-Path $LibAPath) {
    Copy-Item -Force $LibAPath (Join-Path $VoskModSrcDir "libvosk.dll.a")
    Write-Host "    libvosk.dll.a kopiert nach $VoskModSrcDir"
} elseif (Test-Path $LibPath) {
    Copy-Item -Force $LibPath (Join-Path $VoskModSrcDir "libvosk.lib")
    Write-Host "    libvosk.lib kopiert nach $VoskModSrcDir"
}

$env:CGO_ENABLED  = "1"
$env:CC           = "gcc"
$env:CXX          = "g++"
# CGO_CFLAGS / CGO_CPPFLAGS NICHT global setzen – die Vosk-Go-Package
# findet ihren Header jetzt selbst via #cgo CPPFLAGS: -I ${SRCDIR}/../src.
# Globale Includes treffen sonst auch runtime/cgo und brechen den Build.
$env:CGO_CFLAGS   = ""
$env:CGO_CPPFLAGS = ""
# CGO_LDFLAGS: nur Suchpfad, kein -lvosk (Package ergänzt -lvosk selbst).
$LibDirCgo        = $LibDir -replace "\\", "/"
$VoskModSrcDirCgo = $VoskModSrcDir -replace "\\", "/"
$env:CGO_LDFLAGS  = "-L$LibDirCgo -L$VoskModSrcDirCgo"
$env:LIBRARY_PATH = $LibDir
$env:Path         = "$BinDir;$env:Path"

Write-Host "==> Build startet"
Write-Host "    stdout:       $StdOutLog"
Write-Host "    stderr:       $StdErrLog"
Write-Host "    CGO_CFLAGS:   '$env:CGO_CFLAGS'"
Write-Host "    CGO_CPPFLAGS: '$env:CGO_CPPFLAGS'"
Write-Host "    CGO_LDFLAGS:  '$env:CGO_LDFLAGS'"
Write-Host "    VoskModSrc:   $VoskModSrcDir"

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

if ($Run) {
    Write-Host "==> Starte $OutExe"
    & $OutExe
}

function Test-UrlExists {
    param(
        [Parameter(Mandatory = $true)][string]$Url
    )
    try {
        Invoke-WebRequest -Uri $Url -Method Head -UseBasicParsing | Out-Null
        return $true
    } catch {
        return $false
    }
}
