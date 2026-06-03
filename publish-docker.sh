#!/bin/bash
# Docker Image Publishing Script for GoXRay VPN Client
# Version: 1.7.0
# Usage: ./publish-docker.sh [version]

set -e

# Configuration
DOCKER_USERNAME="${DOCKER_USERNAME:-your-dockerhub-username}"
IMAGE_NAME="goxray"
VERSION="${1:-1.7.0}"
LATEST_TAG="latest"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if logged in to Docker Hub
if ! docker info | grep -q "Username"; then
    log_warn "Not logged in to Docker Hub. Please login first:"
    docker login
fi

log_info "Starting Docker image build and publish process..."
log_info "Image: ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}"

# Step 1: Build the image
log_info "Step 1/5: Building Docker image..."
docker build -t ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION} .

if [ $? -ne 0 ]; then
    log_error "Docker build failed!"
    exit 1
fi

log_info "Docker image built successfully!"

# Step 2: Tag as latest
log_info "Step 2/5: Tagging image as latest..."
docker tag ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION} ${DOCKER_USERNAME}/${IMAGE_NAME}:${LATEST_TAG}

# Step 3: Test the image
log_info "Step 3/5: Testing the image..."
docker run --rm ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION} --help > /dev/null 2>&1

if [ $? -ne 0 ]; then
    log_warn "Image test failed, but continuing..."
else
    log_info "Image test passed!"
fi

# Step 4: Push version tag
log_info "Step 4/5: Pushing version tag to Docker Hub..."
docker push ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}

if [ $? -ne 0 ]; then
    log_error "Failed to push version tag!"
    exit 1
fi

log_info "Version tag pushed successfully!"

# Step 5: Push latest tag
log_info "Step 5/5: Pushing latest tag to Docker Hub..."
docker push ${DOCKER_USERNAME}/${IMAGE_NAME}:${LATEST_TAG}

if [ $? -ne 0 ]; then
    log_error "Failed to push latest tag!"
    exit 1
fi

log_info "Latest tag pushed successfully!"

# Summary
echo ""
log_info "=========================================="
log_info "Docker image published successfully!"
log_info "=========================================="
log_info "Image: ${DOCKER_USERNAME}/${IMAGE_NAME}"
log_info "Tags: ${VERSION}, ${LATEST_TAG}"
log_info ""
log_info "Pull command:"
echo "  docker pull ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION}"
echo "  docker pull ${DOCKER_USERNAME}/${IMAGE_NAME}:${LATEST_TAG}"
log_info ""
log_info "Run command:"
echo "  docker run -d --privileged --network host \\"
echo "    -v ./config.yaml:/etc/goxray/config.yaml:ro \\"
echo "    ${DOCKER_USERNAME}/${IMAGE_NAME}:${VERSION} \\"
echo "    --config /etc/goxray/config.yaml"
log_info "=========================================="
