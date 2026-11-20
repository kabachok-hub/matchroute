[CmdletBinding()]
param(
    [string]$Version = 'v0.1.0',
    [string]$Go = 'go',
    [string]$OutputDirectory = 'dist'
)

$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^v\d+\.\d+\.\d+(?:-[a-zA-Z0-9.-]+)?$') {
    throw 'Version must have the form v0.1.0.'
}
$Go = (Get-Command $Go -CommandType Application).Source
$root = Split-Path -Parent $PSScriptRoot
$env:CGO_ENABLED = '0'
$env:GOTOOLCHAIN = 'local'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'

function Invoke-Go {
    & $Go @args
    if ($LASTEXITCODE -ne 0) {
        throw "Go command failed: $($args -join ' ')"
    }
}

function Get-VerifiedArchive($Uri, $Path, $Hash) {
    if (-not (Test-Path -LiteralPath $Path)) {
        Invoke-WebRequest -Uri $Uri -OutFile $Path
    }
    if ((Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash -ne $Hash) {
        throw "SHA-256 mismatch: $Path"
    }
}

Push-Location $PSScriptRoot
try {
    $out = New-Item -ItemType Directory -Path $OutputDirectory -Force
    $cache = New-Item -ItemType Directory -Path 'dist/matchroute-build' -Force
    Set-Content -LiteralPath (Join-Path $PSScriptRoot 'dist/go.mod') -Value 'module fnport.invalid/build-artifacts'
    $bundleName = "MatchRoute-$Version-windows-x64"
    $bundle = New-Item -ItemType Directory -Path (Join-Path $out.FullName $bundleName) -Force
    $fakes = New-Item -ItemType Directory -Path (Join-Path $bundle.FullName 'fakes') -Force
    Copy-Item -Path (Join-Path $root 'fnport/files/quic_initial_*.bin') -Destination 'internal/fakes'
    Copy-Item -Path (Join-Path $root 'fnport/files/quic_initial_*.bin') -Destination $fakes.FullName

    $wintun = Join-Path $cache.FullName 'wintun-0.14.1.zip'
    Get-VerifiedArchive 'https://www.wintun.net/builds/wintun-0.14.1.zip' $wintun '07c256185d6ee3652e09fa55c0b673e2624b565e02c4b9091c79ca7d2f24ef51'
    Expand-Archive -LiteralPath $wintun -DestinationPath $cache.FullName -Force
    $dll = Join-Path $cache.FullName 'wintun/bin/amd64/wintun.dll'
    if ((Get-FileHash -LiteralPath $dll -Algorithm SHA256).Hash -ne 'e5da8447dc2c320edc0fc52fa01885c103de8c118481f683643cacc3220dafce') {
        throw 'Unexpected wintun.dll'
    }
    Copy-Item -LiteralPath $dll -Destination $bundle.FullName

    $revision = 'c005f35300ce76427dc285481ec5695f5b179cd2'
    $coreArchive = Join-Path $cache.FullName "$revision.tar.gz"
    $coreSource = Join-Path $cache.FullName "sing-box-$revision"
    Get-VerifiedArchive "https://codeload.github.com/SagerNet/sing-box/tar.gz/$revision" $coreArchive '1c4d2b3bf261a8d98e1b026be0bbcddbcb3f53bbca77eb15e8f25f7d4e602e59'
    & tar.exe -xzf $coreArchive -C $cache.FullName
    if ($LASTEXITCODE -ne 0) { throw 'Cannot unpack the sing-box source archive.' }
    $core = Join-Path $bundle.FullName 'sing-box.exe'
    Invoke-Go -C $coreSource build -buildvcs=false -trimpath -tags 'with_quic,with_utls,badlinkname,tfogo_checklinkname0' -ldflags '-s -w -buildid= -checklinkname=0 -X github.com/sagernet/sing-box/constant.Version=1.14.4 -X runtime.godebugDefault=multipathtcp=0,tlssha1=1' -o $core ./cmd/sing-box

    $goroot = Invoke-Go env GOROOT
    $licensePaths = @(
        (Join-Path $root 'LICENSE')
        (Join-Path $goroot 'LICENSE')
        'third_party/wintun-LICENSE.txt'
        'third_party/wgcf-LICENSE.txt'
        'internal/fonts/OFL.txt'
        (Join-Path $root 'fnport/files/quic_initial_vk_com.bin.LICENSE')
        (Join-Path $coreSource 'LICENSE')
        'third_party/sing-box-GPL-3.0.txt'
        'third_party/sing-box-SOURCES.txt'
    )
    Set-Content -LiteralPath 'fnport/licenses.txt' -Value '' -Encoding utf8
    $modules = Invoke-Go list -deps -f '{{with .Module}}{{if not .Main}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}{{end}}' ./fnport
    $licenses = foreach ($module in ($modules | Where-Object { $_ } | Sort-Object -Unique)) {
        $name, $modVersion, $dir = $module.Split('|', 3)
        $file = Get-ChildItem -LiteralPath $dir -File | Where-Object Name -Match '^(licen[cs]e|copying|unlicense)' | Select-Object -First 1
        if (-not $file) { throw "Missing license for $name $modVersion" }
        "=== $name $modVersion ==="
        Get-Content -LiteralPath $file.FullName -Raw
    }
    foreach ($path in $licensePaths) {
        $licenses += "=== $(Split-Path -Leaf $path) ==="
        $licenses += Get-Content -LiteralPath $path -Raw
    }
    $licenses | Set-Content -LiteralPath 'fnport/licenses.txt' -Encoding utf8
    Copy-Item -LiteralPath 'fnport/licenses.txt' -Destination (Join-Path $bundle.FullName 'LICENSES.txt')

    Invoke-Go run ./tools/mkicon fnport/winres
    Push-Location fnport
    try {
        Invoke-Go run github.com/tc-hib/go-winres@v0.3.3 make --in winres/winres.json --arch amd64
    } finally {
        Pop-Location
    }
    $env:FNPORT_TEST_SING_BOX = $core
    Invoke-Go vet ./...
    Invoke-Go test -count=1 ./fnport
    Invoke-Go build -trimpath -ldflags "-s -w -H windowsgui -X main.version=$Version -X main.devTrace=off" -o (Join-Path $bundle.FullName 'MatchRoute.exe') ./fnport

    $zip = Join-Path $out.FullName "$bundleName.zip"
    Compress-Archive -LiteralPath $bundle.FullName -DestinationPath $zip -Force
    $digest = Get-FileHash -LiteralPath $zip -Algorithm SHA256
    "$($digest.Hash.ToLowerInvariant())  $bundleName.zip" | Set-Content -LiteralPath (Join-Path $out.FullName 'SHA256SUMS.txt') -Encoding ascii
    Get-Item -LiteralPath $zip
} finally {
    Pop-Location
}
