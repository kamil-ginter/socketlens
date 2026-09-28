Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

. (Join-Path $PSScriptRoot 'env.ps1')

$DevDir = Join-Path $ProjectRoot 'artifacts\dev'
$Exe = Join-Path $DevDir 'SocketLens.exe'
$DataDir = Join-Path $ProjectRoot 'data'

New-Item -ItemType Directory -Force -Path $DevDir | Out-Null
New-Item -ItemType Directory -Force -Path $DataDir | Out-Null

Push-Location $ProjectRoot
try {
    go build -trimpath -o $Exe ./cmd/socketlens
    if ($LASTEXITCODE -ne 0) {
        throw "SocketLens build failed."
    }
}
finally {
    Pop-Location
}

$process = Start-Process `
    -FilePath $Exe `
    -ArgumentList @('--listen', '127.0.0.1:8790', '--data', $DataDir) `
    -WorkingDirectory $ProjectRoot `
    -PassThru

try {
    $deadline = (Get-Date).AddSeconds(12)
    $ready = $false

    while ((Get-Date) -lt $deadline) {
        if ($process.HasExited) {
            throw "SocketLens exited before startup completed."
        }

        try {
            $health = Invoke-RestMethod -Uri 'http://127.0.0.1:8790/api/health' -TimeoutSec 2
            if ($health.status -eq 'healthy') {
                $ready = $true
                break
            }
        }
        catch {
            Start-Sleep -Milliseconds 300
        }
    }

    if (-not $ready) {
        throw "SocketLens did not become ready."
    }

    Start-Process 'http://127.0.0.1:8790'
    Write-Host ""
    Write-Host "SocketLens is running."
    Write-Host "Dashboard: http://127.0.0.1:8790"
    Write-Host "Close this window or press Ctrl+C to stop it."
    Write-Host ""

    $process.WaitForExit()
}
finally {
    if ($process -and -not $process.HasExited) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    }
}
