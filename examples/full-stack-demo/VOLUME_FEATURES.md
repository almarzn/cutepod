# Enhanced Volume Features Demo

This document explains the advanced volume features demonstrated in the full-stack demo.

## Volume Types Demonstrated

### 1. Named Volumes (Podman-managed)
```yaml
# Database persistent storage
apiVersion: cutepod.io/v1
kind: CuteVolume
metadata:
  name: postgres-data
spec:
  type: volume
  volume:
    driver: local
    options:
      type: none
      o: bind
  securityContext:
    owner:
      user: 999   # postgres user
      group: 999  # postgres group
    seLinuxOptions:
      level: private  # Z flag - private access
```

**Usage in Container:**
```yaml
volumes:
  - name: postgres-data
    mountPath: /var/lib/postgresql/data
    readOnly: false
    mountOptions:
      seLinuxLabel: Z  # Private access for database
```

### 2. hostPath Volumes (Host directory mounts)
```yaml
# Shared web content
apiVersion: cutepod.io/v1
kind: CuteVolume
metadata:
  name: webapp-content
spec:
  type: hostPath
  hostPath:
    path: /tmp/webapp-content
    type: DirectoryOrCreate
  securityContext:
    owner:
      user: 101   # nginx user
      group: 101  # nginx group
    seLinuxOptions:
      level: shared  # z flag - shared access
```

**Usage with SubPath:**
```yaml
volumes:
  - name: webapp-content
    mountPath: /usr/share/nginx/html/static
    subPath: static-content  # Mount only the static-content subdirectory
    readOnly: false
    mountOptions:
      seLinuxLabel: z  # Shared access between webapp instances
```

### 3. emptyDir Volumes (Temporary storage)
```yaml
# High-performance cache storage
apiVersion: cutepod.io/v1
kind: CuteVolume
metadata:
  name: redis-cache
spec:
  type: emptyDir
  emptyDir:
    medium: Memory  # Use tmpfs (RAM-backed filesystem)
    sizeLimit: 256Mi
  securityContext:
    owner:
      user: 999   # redis user
      group: 999  # redis group
```

**Usage:**
```yaml
volumes:
  - name: redis-cache
    mountPath: /data
    readOnly: false
    mountOptions:
      seLinuxLabel: Z  # Private access for Redis data
```

## Advanced Mount Features

### SubPath Organization
The demo organizes files using subPath to avoid conflicts:

```
Host Directory: /tmp/app-logs/
├── nginx/
│   ├── instance-1/     # WebApp-1 logs (subPath: nginx/instance-1)
│   └── instance-2/     # WebApp-2 logs (subPath: nginx/instance-2)
├── api/                # API service logs (subPath: api)
├── redis/              # Redis logs (subPath: redis)
└── postgresql/         # Database logs (subPath: postgresql)
```

**Container Mount Examples:**
```yaml
# WebApp Instance 1
volumes:
  - name: app-logs
    mountPath: /var/log/nginx
    subPath: nginx/instance-1
    readOnly: false

# WebApp Instance 2
volumes:
  - name: app-logs
    mountPath: /var/log/nginx
    subPath: nginx/instance-2
    readOnly: false

# API Service
volumes:
  - name: app-logs
    mountPath: /var/log/app
    subPath: api
    readOnly: false
```

### SELinux Integration
The demo demonstrates proper SELinux labeling:

- **`z` (shared)**: Multiple containers can access the same volume
  - Used for: logs, configuration files, shared web content
  - Example: webapp instances sharing static content
  
- **`Z` (private)**: Only one container can access the volume
  - Used for: database data, cache data
  - Example: PostgreSQL data directory, Redis cache

### Read-Only Configuration Mounts
Configuration files are mounted read-only for security:

```yaml
# Configuration file mount
volumes:
  - name: app-config
    mountPath: /etc/nginx/nginx.conf
    subPath: nginx/nginx.conf
    readOnly: true  # Read-only for security
    mountOptions:
      seLinuxLabel: z  # Shared read-only config
```

## Security Context Integration

### Ownership Management
Volumes automatically get proper ownership based on the container's security context:

```yaml
# Volume definition with ownership
securityContext:
  owner:
    user: 101   # nginx user
    group: 101  # nginx group

# Container using the volume
spec:
  securityContext:
    runAsUser: 101
    runAsGroup: 101
```

### Rootless Podman Support
The system automatically handles user namespace mapping for rootless Podman:
- Container UID 101 might map to host UID 100101
- Volume ownership is automatically adjusted
- No manual intervention required

## Volume Dependency Resolution

Volumes are created before containers that reference them:

1. **Volume Creation**: All volumes are created first
2. **Path Preparation**: Host directories are created with proper permissions
3. **Container Creation**: Containers are created with resolved volume paths

## Performance Considerations

### Memory-backed Storage
```yaml
# High-performance cache using tmpfs
emptyDir:
  medium: Memory
  sizeLimit: 256Mi
```
- Stored in RAM for maximum performance
- Automatically cleaned up when container stops
- Perfect for caches and temporary data

### Bind Mount Optimization
```yaml
# Optimized bind mount
hostPath:
  path: /tmp/webapp-content
  type: DirectoryOrCreate
```
- Direct host filesystem access
- No copy overhead
- Shared between multiple containers efficiently

## Troubleshooting

### Common Issues and Solutions

1. **Permission Denied**
   - Check SELinux labels (`z` vs `Z`)
   - Verify ownership settings in securityContext
   - Ensure host directory permissions

2. **SubPath Not Found**
   - Verify the subPath exists in the volume
   - Check that the parent directory was created
   - Ensure proper path separators (forward slashes)

3. **Volume Mount Conflicts**
   - Use different subPaths for different containers
   - Avoid overlapping mount paths within the same container
   - Check for read-only vs read-write conflicts

### Debugging Commands
```bash
# Check volume mounts
podman inspect <container-name> | jq '.Mounts'

# Check SELinux context
ls -Z /tmp/webapp-content/

# Check ownership
ls -la /tmp/app-logs/

# Check container logs
podman logs <container-name>
```

## Best Practices

1. **Use SubPaths**: Organize files within volumes using subPath
2. **Proper SELinux Labels**: Use `z` for shared, `Z` for private access
3. **Read-Only Configs**: Mount configuration files as read-only
4. **Memory for Caches**: Use emptyDir with Memory medium for caches
5. **Ownership Management**: Set proper user/group in securityContext
6. **Path Types**: Use DirectoryOrCreate for flexibility
7. **Size Limits**: Set sizeLimit for emptyDir volumes to prevent resource exhaustion

This comprehensive example demonstrates all the enhanced volume features available in Cutepod!