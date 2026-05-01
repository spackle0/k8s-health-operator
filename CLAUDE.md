# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## About this project

A Kubernetes operator built with [Kubebuilder](https://book.kubebuilder.io) that watches `HealthPolicy` custom resources (group `monitoring.hugh.local`, kind `HealthPolicy`, version `v1alpha1`). The controller observes pods and events across user-specified namespaces, evaluates them against the policy's rules with **deterministic Go code**, and writes findings to the CR status. No mutations.

Built during Recurse Center batch 2 (Mar 30 – May 8, 2026). The owner is learning Go through this project; pace and style reflect that — see *Working style* below.

See `AGENTS.md` for detailed Kubebuilder patterns, CLI cheat sheets, and reference links.

## Scope for this batch

**In scope (detection layer):**
- `HealthPolicy` CRD with namespace selection + rule definitions
- Reconciler that watches pods/events in selected namespaces
- Rule evaluators: `CrashLoopDetection`, `OOMKillDetection`, `PendingPodDetection`
- Findings written to CR status (separate from policy-health `Conditions`)
- Periodic re-evaluation via `reportingInterval`
- Optional AI agent enrichment: when `agentIntegration.enabled`, controller POSTs each finding to a configured endpoint and appends the response to status. Controller must work fine when the agent is absent or down.
- Optional webhook notifications on `FindingCreated` / `FindingResolved`
- Leader election, Prometheus metrics, structured logging, container image, RBAC
- Real workload target: an MLflow deployment on k3s (PostgreSQL backend) — the controller monitors a real AI workload, not toy nginx pods

**Explicitly deferred (do not scaffold):**
- `RemediationPolicy` CRD and the SelfHealer mutating controller — originally Week 4 of the plan; the owner is focusing on detection first. Don't create `remediationpolicy_types.go`, don't add a second controller, don't add mutating action types (`BumpMemoryLimit`, `RestartPod`, etc.).
- Admission webhooks for policy validation — comes with the SelfHealer
- Finalizers, owner references for child resources — none needed yet (no children)
- CRD versioning beyond `v1alpha1` — no conversion webhooks
- Helm chart, multi-cluster, UI dashboard

If a design choice now would make eventual SelfHealer work easier *without* costing complexity today, mention it as an aside. Don't build for it.

## Design philosophy

**Deterministic detection, optional AI enrichment.** The controller's reconciliation path must be pure Go — no LLM calls in the critical path. The agent integration is a side enrichment that runs after detection completes; if it fails or is unconfigured, detection still works and webhooks still fire.

This is a deliberate inversion of HolmesGPT's operator mode (where the LLM *is* the reconciliation). Keep that distinction visible in the code — agent calls happen in a clearly separable client (`internal/controller/agent/`), webhook dispatch in another (`internal/controller/notifier/`), and the reconciler treats both as best-effort post-detection hooks.

**Status discipline.** `Conditions []metav1.Condition` describes the *policy's own* health (is it being reconciled, is it valid). Per-pod observations go in a separate `Findings` (or similar) slice. Don't conflate the two.

**Minimum viable spec, then grow.** Start `HealthPolicySpec` with the smallest field set that supports a working reconcile loop, then add fields rule-by-rule as evaluators come online. Don't model the full plan-doc spec upfront.

## Working style

The owner wants Claude as a **guide, not a code-writer**. Concretely:

- When asked about a piece, give a focused suggestion for *that piece only*. Don't preemptively scaffold adjacent files or features.
- Prefer a small snippet + explanation of *why* over a complete implementation.
- When the owner is stuck, diagnose before rewriting. Ask what they tried.
- Flag Go idioms as they come up (pointer vs value receivers, error handling, interfaces, goroutines/channels) — they're learning the language through this project.
- After something works, you can name the next logical step — but leave the doing to them unless they ask.
- The owner's background is Python/Kubernetes/PostgreSQL. Don't over-explain pods, services, CRDs. *Do* explain controller-runtime internals (managers, predicates, indexers, finalizers) and Go-specific patterns.

Direct quote from the owner: *"I won't be asking you to write the whole thing for me at once. I will be asking that you take the role of a guide in building this with code suggestions and suggestions for next steps, helping me where I struggle and learn golang."*

## Common commands

```bash
make build          # Compile (also runs manifests, generate, fmt, vet)
make test           # Unit tests via envtest (real K8s API + etcd, no cluster needed)
make lint           # Run golangci-lint
make lint-fix       # Auto-fix lint issues
make run            # Run controller locally against current kubeconfig context
make manifests      # Regenerate CRDs/RBAC from +kubebuilder markers
make generate       # Regenerate zz_generated.deepcopy.go
```

To run a single test package:
```bash
# Set KUBEBUILDER_ASSETS first (make test prints the path), then:
go test ./internal/controller/... -run TestName -v
```

## Code generation rules

After editing `api/v1alpha1/*_types.go` or any `+kubebuilder:` markers, always run:
```bash
make manifests generate
```

Never manually edit:
- `api/v1alpha1/zz_generated.deepcopy.go`
- `config/crd/bases/*.yaml`
- `config/rbac/role.yaml`
- `PROJECT`

Never remove `// +kubebuilder:scaffold:*` comments — the CLI injects code at these markers.

## Architecture

```
cmd/main.go                                       Manager entry point; registers controllers
api/v1alpha1/healthpolicy_types.go                HealthPolicy CRD schema (edit this to add fields)
internal/controller/healthpolicy_controller.go    Reconcile() lives here
internal/controller/watcher/                      (planned) shared event/pod watching + rule evaluator
internal/controller/agent/                        (planned) HTTP client for optional AI agent enrichment
internal/controller/notifier/                     (planned) webhook dispatcher for FindingCreated/Resolved
config/                                           Kustomize manifests (mostly generated)
test/e2e/                                         End-to-end tests against a real cluster
```

The `HealthPolicyReconciler` embeds `client.Client` (for K8s API calls) and `*runtime.Scheme`. The `Reconcile` method is called whenever a `HealthPolicy` object changes; it should be idempotent and re-fetch objects before updating them.

Status uses `[]metav1.Condition` for **policy-level** health — prefer standard types (`Available`, `Progressing`, `Degraded`) over custom condition types. **Per-pod findings are a separate field**, not conditions.

## Naming and identifiers

- **Domain `hugh.local`** is a placeholder. The owner doesn't have an org name yet and may change it. Domain change is cheap (touches `PROJECT` and regenerated manifests; not Go imports).
- **Module path `github.com/spackle0/k8s-health-operator`** is the real GitHub path. Renaming this is broader (`go.mod`, every import).
- Keep the two changes separable in any future rename.

## Logging style

Follow Kubernetes logging conventions (capital first letter, no trailing period, active voice):
```go
log := logf.FromContext(ctx)
log.Info("Starting reconciliation")
log.Error(err, "Failed to fetch HealthPolicy", "name", req.Name)
```

## Reference

Full project plan with week-by-week breakdown lives in the owner's Notion:
https://www.notion.so/hughtipping/Recurse-Center-6-Week-Extension-Plan-Mar-30-May-8-2026-3280e5df33fc81f6b24cf690cae37610

The plan doc uses an older domain (`cogium.io`) and repo name (`k8s-controllers`); read through that mismatch — the live project uses `hugh.local` and `k8s-health-operator`.
