# flaggy Helm chart

Deploy [flaggy](https://github.com/getflaggy/flaggy) — a lightweight feature-flag
server (Go, SQLite, zero runtime dependencies) — on Kubernetes.

## Install

```bash
# from source
helm install flaggy ./charts/flaggy

# or once published as an OCI chart
helm install flaggy oci://ghcr.io/getflaggy/charts/flaggy --version <chart-version>
```

flaggy stores its data in a SQLite file on a PersistentVolume (see `persistence`).

## Auth

Admin routes require a master key. Provide one for production:

```bash
# from an existing Secret (recommended)
helm install flaggy ./charts/flaggy \
  --set auth.existingSecret=flaggy-secrets --set auth.existingSecretKey=master-key

# or inline (creates a Secret; avoid committing this)
helm install flaggy ./charts/flaggy --set auth.masterKey=$(openssl rand -hex 32)
```

With neither set, auth is **disabled** (dev mode only).

## Values

| Key | Default | Description |
|---|---|---|
| `image.repository` | `ghcr.io/getflaggy/flaggy` | Image repo |
| `image.tag` | `""` (chart appVersion) | Image tag |
| `replicaCount` | `1` | Single instance (SQLite) |
| `service.type` / `service.port` | `ClusterIP` / `8080` | Service |
| `config.port` | `":8080"` | `FLAGGY_PORT` |
| `config.dbPath` | `/data/flaggy.db` | SQLite path (keep under `/data`) |
| `config.cors` | `"true"` | `FLAGGY_CORS` (`"false"` behind a proxy) |
| `auth.masterKey` | `""` | Inline master key (creates a Secret) |
| `auth.existingSecret` | `""` | Name of an existing Secret holding the key |
| `auth.existingSecretKey` | `master-key` | Key within that Secret |
| `persistence.enabled` | `true` | PVC for the SQLite file |
| `persistence.size` | `1Gi` | PVC size |
| `persistence.storageClass` | `""` | PVC storage class |
| `ingress.enabled` | `false` | Optional ingress |
| `resources` | `{}` | Pod resources |

See [`values.yaml`](values.yaml) for the full list.
