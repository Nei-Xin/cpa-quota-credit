# Build script for PowerShell on Windows
param (
    [string]$Target = "windows",
    [string]$OutDir = "./bin"
)

if (!(Test-Path $OutDir)) {
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
}

Write-Host "Running unit tests..." -ForegroundColor Cyan
go test -v ./...
if ($LASTEXITCODE -ne 0) {
    Write-Error "Tests failed!"
    exit 1
}

$env:CGO_ENABLED = "1"

switch ($Target.ToLower()) {
    "windows" {
        $outFile = Join-Path $OutDir "cpa-quota-credit.dll"
        Write-Host "Building Windows DLL: $outFile" -ForegroundColor Green
        go build -buildmode=c-shared -o $outFile main.go
    }
    "linux" {
        $outFile = Join-Path $OutDir "cpa-quota-credit.so"
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        Write-Host "Building Linux SO: $outFile" -ForegroundColor Green
        go build -buildmode=c-shared -o $outFile main.go
    }
    default {
        Write-Error "Unknown target: $Target. Use 'windows' or 'linux'."
        exit 1
    }
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build completed successfully!" -ForegroundColor Green
} else {
    Write-Error "Build failed!"
}
