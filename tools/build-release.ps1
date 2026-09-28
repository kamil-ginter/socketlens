Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

. (Join-Path $PSScriptRoot 'env.ps1')

$ReleaseDir = Join-Path $ProjectRoot 'artifacts\release'
$Exe = Join-Path $ReleaseDir 'SocketLens.exe'

New-Item -ItemType Directory -Force -Path $ReleaseDir | Out-Null

Push-Location $ProjectRoot
try {
    go test ./...
    if ($LASTEXITCODE -ne 0) {
        throw "Tests failed."
    }

    go build `
        -trimpath `
        -ldflags "-s -w" `
        -o $Exe `
        ./cmd/socketlens

    if ($LASTEXITCODE -ne 0) {
        throw "Release build failed."
    }
}
finally {
    Pop-Location
}

$hash = Get-FileHash -Algorithm SHA256 $Exe

Write-Host ""
Write-Host "SOCKETLENS RELEASE READY"
Write-Host "Executable: $Exe"
Write-Host "SHA256: $($hash.Hash)"
