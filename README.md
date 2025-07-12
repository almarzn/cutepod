
# Cutepod

Cutepod is an ephemeral, Kubernetes-inspired orchestration tool for local container management using Podman. Written in Go, it sits comfortably between `podman-compose` and Kubernetes in complexity and capability. It uses declarative YAML charts to manage containers, secrets, networks, and volumes — and supports extensibility through custom resource transformations.

---

## ✨ Features

- 🧠 **Kubernetes-Inspired Syntax**  
  Uses structured YAML manifests with Go templating, similar to Helm charts.

- ♻️ **Declarative Reconciliation**  
  Automatically applies, upgrades, and reconciles Podman containers based on chart definitions.

- 🔐 **Secret Management**  
  Supports inline secrets and external secret injection via `CuteSecret` and `CuteSecretStore`.

- 🧩 **Extensibility via `CuteExtension`**  
  Define your own resource kinds and transformation logic with containers.

- 🔄 **Restart Recovery**  
  `cutepod reinit` restarts containers as needed after a server or Podman restart.

- 🚫 **Daemonless & Ephemeral**  
  All commands are CLI-based with no long-running processes.

---

## 📦 Concepts

Cutepod charts are structured YAML manifests that define containerized applications using custom kinds:

| Kind              | Description                                                  |
|-------------------|--------------------------------------------------------------|
| `CuteContainer`   | Defines a Podman container, including ports, env, volumes.   |
| `CutePod`         | Groups containers together with restart policy.              |
| `CuteNetwork`     | Defines named bridge networks (Podman-compatible).           |
| `CuteVolume`      | Creates named or hostPath volumes.                           |
| `CuteSecret`      | Opaque secrets injected as Podman secrets.                   |
| `CuteSecretStore` | Refers to external secret sources to be transformed.         |
| `CuteExtension`   | Defines transformation logic for custom resource kinds.      |

---

## 📂 Chart Structure

Charts use Go templates and follow this directory layout:

```

my-chart/
├── templates/
│   ├── container.yaml
│   ├── secret.yaml
│   └── ...
├── values.yaml
├── chart.yaml

````

**Templates** use Go templating and are rendered using values from `values.yaml`.

---

## 🛠 Installation & Commands

All Cutepod actions are ephemeral and executed via the CLI:

```bash
cutepod lint <path-to-chart>           # Validate templates and YAML structure
cutepod install <namespace> <chart>    # Install containers (use --dry-run to preview)
cutepod upgrade <namespace> <chart>    # Reconcile and update containers (use --dry-run)
cutepod reinit [namespace]             # Restart containers after system/podman restart
````

Use `--dry-run` with `install` or `upgrade` to preview changes without applying them.

---

## 🔐 Secrets and SecretStores

Cutepod supports two types of secret resources:

### Example: CuteSecret

```yaml
kind: CuteSecret
metadata:
  name: db-creds
  namespace: demo
spec:
  type: Opaque
  data:
    username: YWRtaW4=
    password: c3VwZXJzZWNyZXQ=
```

### Example: CuteSecretStore (External)

```yaml
kind: CuteSecretStore
metadata:
  name: from-file
  namespace: demo
spec:
  provider: file
  path: /etc/secrets/db.yaml
```

Secrets are injected as Podman secrets, available via env or mounted file in `CuteContainer`.

---

## 🧩 Extending with CuteExtension

`CuteExtension` lets you define a transformer container that converts custom resources into supported kinds.

### Example: CuteExtension

```yaml
kind: CuteExtension
metadata:
  name: external-to-secret
  namespace: demo
spec:
  targetKind: CuteExternalSecret
  inputSchema: schemas/ext-secret.json
  runner:
    image: myorg/external-secret-transformer:latest
```

The `runner` takes the input object as stdin and must return a valid YAML manifest on stdout.

---

## 🧪 Resource Examples

Here’s a set of minimal examples per kind:

### CuteContainer

```yaml
kind: CuteContainer
metadata:
  name: web
  namespace: demo
spec:
  image: nginx:1.25
  ports:
    - containerPort: 80
  env:
    - name: MODE
      value: production
  volumeMounts:
    - name: static-data
      mountPath: /usr/share/nginx/html
  secrets:
    - name: db-creds
```

### CutePod

```yaml
kind: CutePod
metadata:
  name: web-pod
  namespace: demo
spec:
  containers:
    - name: web
  restartPolicy: Always
```

### CuteVolume

```yaml
kind: CuteVolume
metadata:
  name: static-data
  namespace: demo
spec:
  driver: local
  type: volume
  options:
    o: bind
    device: /host/data
    type: none
```

### CuteNetwork

```yaml
kind: CuteNetwork
metadata:
  name: bridge-net
  namespace: demo
spec:
  driver: bridge
  options:
    - name: subnet
      value: 10.89.0.0/16
```

---

## 🔄 Kubernetes Compatibility

Cutepod is **Kubernetes-inspired**, not compatible. It mimics some structures but is tailored for Podman.

| Kubernetes Concept | Cutepod Equivalent  | Notes                         |
| ------------------ | ------------------- | ----------------------------- |
| Pod                | `CutePod`           | Uses Podman containers        |
| Deployment         | N/A                 | No rolling updates/scheduling |
| Secret (Opaque)    | `CuteSecret`        | Base64 format                 |
| ConfigMap          | ❌ Not yet supported | Planned via extension         |
| Helm chart         | Cutepod chart       | Go template supported         |
| CRD                | `CuteExtension`     | Plugin-style transformations  |
| DaemonSet          | ❌ Not supported     | Not applicable to Podman      |

---

## ⚙️ Requirements

* **Go 1.24+**
* **Podman installed and accessible via `$PATH`**
* **libpod** via `containers/libpod`
* Linux/macOS (rootless Podman supported)
* CLI framework: Cobra (github.com/spf13/cobra) – chosen for idiomatic Go CLI development

---

## 📁 .codex Directory (Autonomous Agent Workspace)

This project is set up for autonomous development via Codex.

* Project planning and milestones tracked in `.codex/plan_<date>.md`
* Logs and changelogs recorded in `.codex/`
* `README.md` is the single source of truth for functionality and behavior.

---

## 📜 Changelog

*Add changelog entries below as Codex progresses with implementation.*

```
### 2025-07-12
- Initial Cutepod README created
- Defined chart structure, kinds, usage and design principles
- Created initial project plan at .codex/plan_2025-07-12.md
- Bootstrapped Go module and Cobra-based CLI entrypoint with command stubs
- Completed Phase 2: Chart Rendering & Validation
- Added initial example chart under examples/demo-chart
```

---

## 📬 Contributing

* Contributions to core kinds, CLI behavior, or extension mechanics are welcome.
* You may define new `CuteExtension` kinds to enhance container transformation workflows.
