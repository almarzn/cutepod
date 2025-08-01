# Full Stack Demo Chart

This comprehensive demo chart showcases all Cutepod resource types and demonstrates proper dependency management, resource orchestration, and real-world application patterns.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Demo Network (172.20.0.0/16)            │
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   WebApp-1  │    │   WebApp-2  │    │             │     │
│  │   :8080     │    │   :8081     │    │   API       │     │
│  │   (nginx)   │    │   (nginx)   │    │   :3000     │     │
│  └─────────────┘    └─────────────┘    │   (node)    │     │
│         │                   │          └─────────────┘     │
│         └───────────────────┼─────────────────┘            │
│                             │                              │
│  ┌─────────────┐    ┌─────────────┐                       │
│  │ PostgreSQL  │    │   Redis     │                       │
│  │   :5432     │    │   :6379     │                       │
│  │ (database)  │    │  (cache)    │                       │
│  └─────────────┘    └─────────────┘                       │
└─────────────────────────────────────────────────────────────┘

External Volumes:
├── postgres-data (named volume with private SELinux access)
├── webapp-content (hostPath: /tmp/webapp-content with shared access)
├── app-logs (hostPath: /tmp/app-logs with subPath organization)
├── redis-cache (emptyDir tmpfs: 256Mi memory-backed)
└── app-config (hostPath: /tmp/app-config with read-only configs)

Secrets:
├── db-credentials (database connection info)
├── api-keys (API configuration and JWT secrets)
└── tls-certs (SSL/TLS certificates)
```

## Resource Types Demonstrated

### 🌐 Network
- **demo-network**: Custom bridge network with subnet `172.20.0.0/16`
- Enables container-to-container communication
- Isolated from host network for security

### 📦 Volumes (Enhanced Features)
- **postgres-data**: Named Podman volume with private SELinux access and proper ownership
- **webapp-content**: hostPath volume with shared SELinux access and subPath support
- **app-logs**: hostPath volume with shared access for centralized logging with subPath organization
- **redis-cache**: emptyDir volume using tmpfs (Memory) for high-performance caching
- **app-config**: hostPath volume for configuration files with read-only access

### 🔐 Secrets
- **db-credentials**: Database connection credentials
- **api-keys**: API keys, JWT secrets, and service URLs
- **tls-certs**: SSL/TLS certificates for secure communication

### 🐳 Containers
- **postgres-db**: PostgreSQL 15 database with persistent storage
- **redis-cache**: Redis 7 cache with memory optimization
- **api-service**: Node.js API service with health checks
- **webapp-1**: Nginx web server instance 1 (port 8080)
- **webapp-2**: Nginx web server instance 2 (port 8081)

## Dependency Chain

The chart demonstrates proper dependency ordering:

```
1. Network (demo-network)
   ↓
2. Volumes (postgres-data, webapp-content, app-logs, redis-cache, app-config)
   ↓
3. Secrets (db-credentials, api-keys, tls-certs)
   ↓
4. Backend Services (postgres-db, redis-cache)
   ↓
5. API Service (api-service) - depends on database and cache
   ↓
6. Frontend Services (webapp-1, webapp-2) - depend on API
```

## Features Showcased

### 🔄 Resource Management
- **Dependency Resolution**: Proper creation order based on dependencies
- **Health Checks**: All containers include health check configurations
- **Resource Limits**: CPU and memory limits for all containers
- **Security Context**: Non-root users and proper permissions

### 🔧 Configuration Management
- **Environment Variables**: Both direct values and secret references
- **Enhanced Volume Mounts**: Multiple volume types with advanced features
  - **hostPath volumes**: With DirectoryOrCreate type and subPath support
  - **Named volumes**: Podman-managed persistent storage
  - **emptyDir volumes**: Memory-backed temporary storage with size limits
  - **SELinux Integration**: Proper labeling (z/Z) for shared/private access
  - **Ownership Management**: Automatic UID/GID mapping for rootless Podman
  - **SubPath Support**: Mount specific subdirectories within volumes
- **Network Configuration**: Custom network with proper subnet configuration
- **Port Mapping**: Both fixed and dynamic port assignments
- **Restart Policies**: Uses Podman-compatible values (`always`, `on-failure`, `no`, `unless-stopped`)

### 🛡️ Security Features
- **Secret Management**: Sensitive data stored in secrets
- **Network Isolation**: Custom network for container communication
- **User Security**: Non-root execution for all containers
- **TLS Support**: SSL certificates mounted for secure communication

## Usage

### Setup Environment
```bash
# Prepare host directories and static content
./examples/full-stack-demo/setup.sh
```

### Install the Demo
```bash
# Dry run to see what will be created
cutepod install demo examples/full-stack-demo --dry-run

# Install the full stack
cutepod install demo examples/full-stack-demo

# Check status
cutepod status demo
```

### Access the Applications
- **Web App Instance 1**: http://localhost:8080
- **Web App Instance 2**: http://localhost:8081
- **API Service**: http://localhost:3000

### Upgrade the Demo
```bash
# Modify values.yaml and upgrade
cutepod upgrade demo examples/full-stack-demo --dry-run
cutepod upgrade demo examples/full-stack-demo
```

### Clean Up
```bash
# Clean up all resources (containers, volumes, secrets, networks)
./examples/full-stack-demo/cleanup.sh

# Or use cutepod uninstall (when implemented)
# cutepod uninstall demo
```

## Customization

Edit `values.yaml` to customize:
- **Images**: Change container images and versions (use `docker.io/library/` prefix)
- **Resources**: Adjust CPU/memory limits
- **Network**: Modify subnet and network configuration
- **Ports**: Change port mappings
- **Secrets**: Update credentials and API keys

### Restart Policy Options
Containers support the following restart policies:
- `no`: Do not restart containers on exit
- `on-failure[:max_retries]`: Restart on non-zero exit code
- `always`: Restart containers when they exit, regardless of status
- `unless-stopped`: Identical to always

## Monitoring

The demo includes:
- **Health Checks**: All containers have health check endpoints
- **Logging**: Centralized logging to `/tmp/app-logs`
- **Resource Monitoring**: Resource limits and requests configured
- **Status Endpoints**: API service provides status information

## Enhanced Volume Features Demonstrated

This demo showcases the advanced volume capabilities:

### Volume Types
- **Named Volumes**: `postgres-data` uses a Podman-managed volume for persistence
- **hostPath Volumes**: `webapp-content`, `app-logs`, and `app-config` use host directory mounts
- **emptyDir Volumes**: `redis-cache` uses memory-backed temporary storage

### Advanced Mount Options
- **SubPath Support**: Different containers mount different subdirectories
  - Logs are organized: `nginx/instance-1`, `nginx/instance-2`, `api`, `redis`, `postgresql`
  - Config files: `nginx/nginx.conf`, `api/config.json`
  - Static content: `static-content` subdirectory
- **SELinux Labels**: Proper security labeling for shared vs private access
  - `z` (shared): Multiple containers can access (logs, configs, webapp content)
  - `Z` (private): Single container access (database data, cache data)
- **Read-Only Mounts**: Configuration files mounted read-only for security

### Security Context Integration
- **Ownership Management**: Volumes automatically get proper UID/GID ownership
  - Database: `999:999` (postgres user)
  - Web servers: `101:101` (nginx user)
  - API service: `1000:1000` (node user)
  - Cache: `999:999` (redis user)
- **Rootless Podman Support**: Automatic user namespace mapping

### Volume Organization
```
/tmp/webapp-content/
└── static-content/          # Shared by both webapp instances

/tmp/app-logs/
├── nginx/
│   ├── instance-1/          # WebApp-1 logs
│   └── instance-2/          # WebApp-2 logs
├── api/                     # API service logs
├── redis/                   # Redis logs
└── postgresql/              # Database logs

/tmp/app-config/
├── nginx/
│   └── nginx.conf           # Nginx configuration
└── api/
    └── config.json          # API configuration

Memory (tmpfs):
└── redis-cache (256Mi)      # High-performance cache storage
```

## Educational Value

This demo teaches:
1. **Dependency Management**: How to structure complex applications
2. **Resource Types**: Usage of all Cutepod resource types
3. **Security Practices**: Proper secret and security configuration
4. **Networking**: Container networking and communication
5. **Enhanced Storage**: Advanced volume types, subPath, and security features
6. **Scaling**: Multiple container instances with load balancing
7. **Configuration**: Environment-based configuration management
8. **SELinux Integration**: Proper security labeling for container volumes
9. **Rootless Podman**: User namespace mapping and ownership management

Perfect for learning Cutepod's capabilities and best practices!