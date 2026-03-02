# Copilot Instructions

## Project Overview

A Kubernetes controller that bridges Kubernetes **Ingress** and **Gateway API HTTPRoute** resources with [Gatus](https://github.com/TwiN/gatus) (a status monitoring tool). It watches cluster resources and automatically generates Gatus endpoint monitoring configurations, aggregated into a ConfigMap.

## Commands

Tools are managed via [mise](https://mise.jdx.dev/) (`mise.toml`) and tasks via [Task](https://taskfile.dev/) (`Taskfile.yml`). Run `mise install` to install all required tools.

```bash
task build        # go build ./...
task test         # go test ./... -v
task fmt          # go fmt ./...
task vet          # go vet ./...
task docker-build # Build container image (IMAGE_REGISTRY/IMAGE_REPO:IMAGE_TAG)
task hooks:install # Install pre-commit hooks (run once after mise install)
task e2e          # Run e2e tests — requires a configured KUBECONFIG (e.g. k3s). Usage: task e2e TAG=ci-test
```

Run a single test:
```bash
go test ./internal/controller/... -run TestSanitizeHostname -v
```

Validate GitHub Actions locally with [act](https://github.com/nektos/act) (requires Docker and `gh` CLI authenticated):
```bash
task act:pr                   # go + docker jobs of pr.yml (e2e skipped — needs real k3s)
task act:main                 # go job of main.yml (docker job pushes to registry, skipped)
task act:tag TAG=v1.2.3       # full tag workflow_dispatch; docker job needs existing SHA image in registry
task act:tag                  # uses v0.0.0-test as default tag
```

## Architecture

**Data flow**: Ingress/HTTPRoute → `GatusEndpoint` CR → aggregated into ConfigMap (`gatus-config/endpoints.yaml`) → Gatus reads config.

**Five reconcilers** registered in `cmd/main.go`:
- `IngressReconciler` — watches `networking.k8s.io/v1` Ingresses, creates/updates/deletes `GatusEndpoint` CRs
- `HTTPRouteReconciler` — opt-in Gateway API support (enabled via `GATEWAY_API_ENABLED=true`), same pattern as ingress
- `GatusEndpointReconciler` — aggregates all `GatusEndpoint` CRs into a single ConfigMap in `TARGET_NAMESPACE`
- `GatusExternalEndpointReconciler` — handles manually-defined external endpoint monitoring
- `GatusAlertReconciler` — validates `GatusAlert` CRs (alert provider configurations)

**Custom resources** defined in `api/v1alpha1/`:
- `GatusEndpoint` — HTTP/DNS/SSH endpoint monitoring spec
- `GatusAlert` — alert provider configuration
- `GatusExternalEndpoint` — manually-defined external resources

## Key Conventions

### Annotations (on Ingress/HTTPRoute)
| Annotation | Effect |
|---|---|
| `gatus.io/enabled: "false"` | Disables monitoring (deletes existing `GatusEndpoint`) |
| `gatus.io/group` | Groups endpoints in the Gatus UI (default: `"external"`) |
| `gatus.io/alerts` | Comma-separated list of `GatusAlert` names to attach |

### Labels (on managed resources)
- `gatus.io/managed-by: gatus-ingress-controller` — marks controller-owned resources
- `gatus.io/ingress: <name>` — links a `GatusEndpoint` to its source Ingress

### Hostname sanitization
Endpoint names are derived from `<ingress-name>-<sanitized-host>`: dots become dashes, wildcards (`*`) become `wildcard`. Example: `my-ingress-wildcard-example-com`.

### Runtime configuration (env vars)
| Variable | Default | Purpose |
|---|---|---|
| `INGRESS_CLASS` | `traefik` | Only reconcile Ingresses with this class |
| `TARGET_NAMESPACE` | `gatus` | Namespace where the aggregated ConfigMap is written |
| `GATEWAY_API_ENABLED` | `false` | Enable HTTPRoute controller (CRDs must be installed) |
| `GATEWAY_NAMES` | _(all)_ | Comma-separated list to filter HTTPRoutes by parent gateway |

### Error handling
Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`.

### Logging
Use `log.FromContext(ctx)` — never initialize a new logger directly in reconcilers.

### Tests
Unit tests use `controller-runtime`'s `fake.Client` (no envtest/cluster required). Test scheme setup follows the `newTestScheme(t *testing.T)` helper pattern in each test file.
