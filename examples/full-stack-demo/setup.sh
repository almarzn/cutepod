#!/bin/bash

# Full Stack Demo Setup Script
# This script prepares the host environment for the demo

set -e

echo "🚀 Setting up Full Stack Demo environment..."

# Create required host directories with enhanced structure
echo "📁 Creating host directories..."
mkdir -p /tmp/webapp-content/static-content
mkdir -p /tmp/app-logs/{nginx/{instance-1,instance-2},api,redis,postgresql}
mkdir -p /tmp/app-config/{nginx,api}

# Set proper permissions
echo "🔐 Setting permissions..."
chown -R $USER:$USER /tmp/webapp-content
chown -R $USER:$USER /tmp/app-logs
chown -R $USER:$USER /tmp/app-config
chmod -R 755 /tmp/webapp-content
chmod -R 755 /tmp/app-logs
chmod -R 755 /tmp/app-config

# Create sample static content
echo "📄 Creating sample static content..."
cat > /tmp/webapp-content/static-content/demo.html << 'EOF'
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

# Create configuration files
echo "⚙️ Creating configuration files..."
cat > /tmp/app-config/nginx/nginx.conf << 'EOF'
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';
    
    access_log /var/log/nginx/access.log main;
    
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    
    server {
        listen 80;
        server_name localhost;
        
        location / {
            root /usr/share/nginx/html;
            index index.html index.htm;
        }
        
        location /static/ {
            alias /usr/share/nginx/html/static/;
            expires 1d;
            add_header Cache-Control "public, immutable";
        }
        
        error_page 500 502 503 504 /50x.html;
        location = /50x.html {
            root /usr/share/nginx/html;
        }
    }
}
EOF

cat > /tmp/app-config/api/config.json << 'EOF'
{
  "server": {
    "port": 3000,
    "host": "0.0.0.0"
  },
  "database": {
    "host": "postgres-db",
    "port": 5432,
    "name": "demodb",
    "pool": {
      "min": 2,
      "max": 10
    }
  },
  "cache": {
    "host": "redis-cache",
    "port": 6379,
    "ttl": 3600
  },
  "logging": {
    "level": "info",
    "file": "/var/log/app/api.log"
  }
}
EOF

cat > /tmp/webapp-content/static-content/api-docs.json << 'EOF'
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
echo "   • Host directories created with enhanced structure:"
echo "     - /tmp/webapp-content/static-content/"
echo "     - /tmp/app-logs/{nginx,api,redis,postgresql}/"
echo "     - /tmp/app-config/{nginx,api}/"
echo "   • Sample static content and configurations created"
echo "   • Permissions configured for enhanced volume features"
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