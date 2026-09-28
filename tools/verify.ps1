Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

. (Join-Path $PSScriptRoot 'env.ps1')

$Artifacts = Join-Path $ProjectRoot 'artifacts'
$VerifyDir = Join-Path $Artifacts 'verify'
$Exe = Join-Path $VerifyDir 'SocketLens.exe'
$SmokeData = Join-Path $VerifyDir 'smoke-data'
$Stdout = Join-Path $VerifyDir 'socketlens.out.log'
$Stderr = Join-Path $VerifyDir 'socketlens.err.log'

function Need([string]$Name) {
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "$Name is required."
    }
}

function Run([string]$Title, [scriptblock]$Action) {
    Write-Host ""
    Write-Host "[$Title]"
    & $Action
    if ($LASTEXITCODE -ne 0) {
        throw "$Title failed with exit code $LASTEXITCODE."
    }
}

Write-Host "=== SocketLens verification ==="
Write-Host "Project: $ProjectRoot"
Write-Host "Go modules: $env:GOMODCACHE"
Write-Host "Go cache: $env:GOCACHE"

Need 'go'
Need 'git'

New-Item -ItemType Directory -Force -Path $VerifyDir | Out-Null

Push-Location $ProjectRoot
try {
    $unformatted = @(gofmt -l .)
    if ($unformatted.Count -gt 0) {
        $unformatted | ForEach-Object { Write-Host "UNFORMATTED: $_" }
        throw "Go formatting gate failed."
    }

    Run '1/5 Go vet' {
        go vet ./...
    }

    Run '2/5 Go tests' {
        go test ./...
    }

    Run '3/5 Go build' {
        go build -trimpath -o $Exe ./cmd/socketlens
    }
}
finally {
    Pop-Location
}

Write-Host ""
Write-Host "[4/5 Runtime + embedded UI smoke test]"

if (Test-Path $SmokeData) {
    Remove-Item $SmokeData -Recurse -Force
}
if (Test-Path $Stdout) {
    Remove-Item $Stdout -Force
}
if (Test-Path $Stderr) {
    Remove-Item $Stderr -Force
}

$process = Start-Process `
    -FilePath $Exe `
    -ArgumentList @('--listen', '127.0.0.1:8791', '--data', $SmokeData) `
    -WorkingDirectory $ProjectRoot `
    -RedirectStandardOutput $Stdout `
    -RedirectStandardError $Stderr `
    -PassThru `
    -WindowStyle Hidden

try {
    $deadline = (Get-Date).AddSeconds(15)
    $health = $null

    while ((Get-Date) -lt $deadline) {
        if ($process.HasExited) {
            $out = if (Test-Path $Stdout) { Get-Content $Stdout -Raw } else { '' }
            $err = if (Test-Path $Stderr) { Get-Content $Stderr -Raw } else { '' }
            throw "SocketLens exited during startup.`n$out`n$err"
        }

        try {
            $health = Invoke-RestMethod -Uri 'http://127.0.0.1:8791/api/health' -TimeoutSec 2
            break
        }
        catch {
            Start-Sleep -Milliseconds 300
        }
    }

    if (-not $health -or $health.status -ne 'healthy' -or $health.version -ne '0.1.0') {
        throw "Health check failed."
    }

    $page = Invoke-WebRequest -Uri 'http://127.0.0.1:8791/' -TimeoutSec 3 -UseBasicParsing
    if ($page.StatusCode -ne 200 -or -not $page.Content.Contains('SocketLens')) {
        throw "Embedded dashboard smoke test failed."
    }

    Write-Host "RUNTIME=PASS"
    Write-Host "EMBEDDED_UI=PASS"

    Write-Host ""
    Write-Host "[5/5 End-to-end local scan + persistence]"

    $payload = @{
        target = '127.0.0.1'
        ports = '8791'
        timeoutMs = 500
    } | ConvertTo-Json

    $scan = Invoke-RestMethod `
        -Uri 'http://127.0.0.1:8791/api/scan' `
        -Method Post `
        -ContentType 'application/json' `
        -Body $payload `
        -TimeoutSec 10

    if ($scan.openPorts.Count -ne 1 -or $scan.openPorts[0].port -ne 8791) {
        throw "End-to-end scan did not detect the SocketLens smoke-test port."
    }

    $history = Invoke-RestMethod -Uri 'http://127.0.0.1:8791/api/history?limit=10' -TimeoutSec 5
    if ($history.Count -lt 1) {
        throw "History persistence smoke test failed."
    }

    Write-Host "PRIVATE_SCAN=PASS"
    Write-Host "SQLITE_HISTORY=PASS"
}
finally {
    if ($process -and -not $process.HasExited) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        $process.WaitForExit()
    }
}

Write-Host ""
Write-Host "=============================================================================="
Write-Host "SOCKETLENS VERIFIED"
Write-Host "GO_VET=PASS"
Write-Host "GO_TESTS=PASS"
Write-Host "GO_BUILD=PASS"
Write-Host "RUNTIME=PASS"
Write-Host "EMBEDDED_UI=PASS"
Write-Host "PRIVATE_SCAN=PASS"
Write-Host "SQLITE_HISTORY=PASS"
Write-Host "=============================================================================="
