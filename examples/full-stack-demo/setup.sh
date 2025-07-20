#!/bin/bash

# Full Stack Demo Setup Script
# This script prepares the host environment for the demo

set -e

echo "🚀 Setting up Full Stack Demo environment..."

# Create required host directories
echo "📁 Creating host directories..."
mkdir -p /tmp/webapp-content
mkdir -p /tmp/app-logs

# Set proper permissions
echo "🔐 Setting permissions..."
chown -R $USER:$USER /tmp/webapp-content
chown -R $USER:$USER /tmp/app-logs
chmod -R 755 /tmp/webapp-content
chmod -R 755 /tmp/app-logs

# Create sample static content
echo "📄 Creating sample static content..."
cat > /tmp/webapp-content/demo.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Static Demo Content</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f8f9fa; }
        .container { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { color: #333; border-bottom: 2px solid #6f42c1; padding-bottom: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="header">Static Content Demo</h1>
        <p>This is static content served from the webapp-content volume mount.</p>
        <p>Location: <code>/tmp/webapp-content/demo.html</code></p>
        <p>This demonstrates how bind mounts work in Cutepod.</p>
    </div>
</body>
</html>
EOF

cat > /tmp/webapp-content/api-docs.json << 'EOF'
{
  "openapi": "3.0.0",
  "info": {
    "title": "Full Stack Demo API",
    "version": "1.0.0",
    "description": "Demo API for Cutepod full stack example"
  },
  "servers": [
    {
      "url": "http://localhost:3000",
      "description": "Demo API Server"
    }
  ],
  "paths": {
    "/": {
      "get": {
        "summary": "Get API status",
        "responses": {
          "200": {
            "description": "API status information",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "message": {"type": "string"},
                    "timestamp": {"type": "string"},
                    "version": {"type": "string"},
                    "environment": {"type": "string"},
                    "database": {"type": "string"},
                    "cache": {"type": "string"}
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}
EOF

echo "✅ Setup complete!"
echo ""
echo "📋 Summary:"
echo "   • Host directories created: /tmp/webapp-content, /tmp/app-logs"
echo "   • Sample static content created"
echo "   • Permissions configured"
echo ""
echo "🚀 Ready to deploy:"
echo "   cutepod install demo examples/full-stack-demo --dry-run"
echo "   cutepod install demo examples/full-stack-demo"
echo ""
echo "🌐 Access URLs (after deployment):"
echo "   • Web App 1: http://localhost:8080"
echo "   • Web App 2: http://localhost:8081"
echo "   • API Service: http://localhost:3000"
echo "   • Static Content: http://localhost:8080/static/demo.html"