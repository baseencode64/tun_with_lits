# Docker Image Publishing Script for GoXRay VPN Client (PowerShell)
# Version: 1.7.0
# Usage: .\publish-docker.ps1 [-Version "1.7.0"] [-Username "your-dockerhub-username"]

param(
    [string]$Version = "1.7.0",
    [string]$Username = $env:DOCKER_USERNAME
)

# Configuration
$ImageName = "goxray"
$LatestTag = "latest"

# Colors for output
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Check if Docker is installed
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Error-Custom "Docker is not installed. Please install Docker Desktop first."
    exit 1
}

# Check if username is provided
if ([string]::IsNullOrEmpty($Username)) {
    Write-Error-Custom "Docker Hub username not provided!"
    Write-Host "Usage: .\publish-docker.ps1 -Username 'your-dockerhub-username' [-Version '1.7.0']"
    Write-Host "Or set environment variable: `$env:DOCKER_USERNAME = 'your-dockerhub-username'"
    exit 1
}

# Check if logged in to Docker Hub
$dockerInfo = docker info 2>&1 | Out-String
if ($dockerInfo -notmatch "Username") {
    Write-Warn "Not logged in to Docker Hub. Attempting login..."
    docker login
    if ($LASTEXITCODE -ne 0) {
        Write-Error-Custom "Docker login failed!"
        exit 1
    }
}

Write-Info "Starting Docker image build and publish process..."
Write-Info "Image: ${Username}/${ImageName}:${Version}"

# Step 1: Build the image
Write-Info "Step 1/5: Building Docker image..."
docker build -t "${Username}/${ImageName}:${Version}" .

if ($LASTEXITCODE -ne 0) {
    Write-Error-Custom "Docker build failed!"
    exit 1
}

Write-Info "Docker image built successfully!"

# Step 2: Tag as latest
Write-Info "Step 2/5: Tagging image as latest..."
docker tag "${Username}/${ImageName}:${Version}" "${Username}/${ImageName}:${LatestTag}"

if ($LASTEXITCODE -ne 0) {
    Write-Error-Custom "Failed to tag image!"
    exit 1
}

# Step 3: Test the image
Write-Info "Step 3/5: Testing the image..."
docker run --rm "${Username}/${ImageName}:${Version}" --help > $null 2>&1

if ($LASTEXITCODE -ne 0) {
    Write-Warn "Image test failed, but continuing..."
} else {
    Write-Info "Image test passed!"
}

# Step 4: Push version tag
Write-Info "Step 4/5: Pushing version tag to Docker Hub..."
docker push "${Username}/${ImageName}:${Version}"

if ($LASTEXITCODE -ne 0) {
    Write-Error-Custom "Failed to push version tag!"
    exit 1
}

Write-Info "Version tag pushed successfully!"

# Step 5: Push latest tag
Write-Info "Step 5/5: Pushing latest tag to Docker Hub..."
docker push "${Username}/${ImageName}:${LatestTag}"

if ($LASTEXITCODE -ne 0) {
    Write-Error-Custom "Failed to push latest tag!"
    exit 1
}

Write-Info "Latest tag pushed successfully!"

# Summary
Write-Host ""
Write-Info "=========================================="
Write-Info "Docker image published successfully!"
Write-Info "=========================================="
Write-Info "Image: ${Username}/${ImageName}"
Write-Info "Tags: ${Version}, ${LatestTag}"
Write-Host ""
Write-Info "Pull command:"
Write-Host "  docker pull ${Username}/${ImageName}:${Version}"
Write-Host "  docker pull ${Username}/${ImageName}:${LatestTag}"
Write-Host ""
Write-Info "Run command:"
Write-Host "  docker run -d --privileged --network host \"
Write-Host "    -v ./config.yaml:/etc/goxray/config.yaml:ro \"
Write-Host "    ${Username}/${ImageName}:${Version} \"
Write-Host "    --config /etc/goxray/config.yaml"
Write-Info "=========================================="
