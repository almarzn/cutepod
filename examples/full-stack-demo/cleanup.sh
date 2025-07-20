#!/bin/bash

# Full Stack Demo Cleanup Script
# This script removes all demo resources to allow for a clean reinstall

set -e

echo "🧹 Cleaning up Full Stack Demo resources..."

# Stop and remove containers
echo "🐳 Removing containers..."
podman rm -f postgres-db redis-cache api-service webapp-1 webapp-2 2>/dev/null || true

# Remove volumes
echo "📦 Removing volumes..."
podman volume rm postgres-data webapp-content app-logs 2>/dev/null || true

# Remove secrets
echo "🔐 Removing secrets..."
podman secret rm db-credentials api-keys tls-certs 2>/dev/null || true

# Remove network
echo "🌐 Removing network..."
podman network rm demo-network 2>/dev/null || true

# Clean up host directories
echo "📁 Cleaning up host directories..."
sudo rm -rf /tmp/webapp-content /tmp/app-logs 2>/dev/null || true

echo "✅ Cleanup complete!"
echo ""
echo "🚀 Ready for fresh deployment:"
echo "   ./examples/full-stack-demo/setup.sh"
echo "   cutepod install demo examples/full-stack-demo"