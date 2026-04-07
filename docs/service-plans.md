# Service plans

Service plans are operator-side presets for `OpenClawInstance.spec.plan`.

They let platform operators publish named, reviewable instance baselines such as:

- CPU and memory requests/limits
- persistent storage size
- default OpenClaw config fragments
- which parts remain overridable per instance

A plan is resolved by the operator from `SERVICE_PLANS_JSON` (rendered from Helm `servicePlans`) and then merged into the instance spec during reconciliation.

## Why use a service plan?

Use a plan when you want a stable, queryable product/profile name instead of repeating the same resource and config defaults in every manifest.

Example:

```yaml
apiVersion: openclaw.rocks/v1alpha1
kind: OpenClawInstance
metadata:
  name: architect-bot
spec:
  plan: architect-juno
  envFrom:
    - secretRef:
        name: openclaw-api-keys
```

The instance may still provide explicit overrides for fields that the selected plan marks as overridable.

## Querying available plans

The operator stores the plan registry in the `SERVICE_PLANS_JSON` environment variable.

Examples:

```bash
# show the raw plan registry configured on the operator deployment
kubectl -n openclaw-system get deploy openclaw-operator \
  -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="SERVICE_PLANS_JSON")].value}' | jq .
```

```bash
# list only the plan names
kubectl -n openclaw-system get deploy openclaw-operator \
  -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="SERVICE_PLANS_JSON")].value}' \
  | jq -r 'keys[]'
```

Adjust the namespace/deployment name to match your installation.

## Current catalog

## `architect-juno`

**Intent:** first real Architect-Bot baseline for serious interactive product and architecture work.

### Baseline resources

- CPU request: `1`
- CPU limit: `4`
- memory request: `4Gi`
- memory limit: `8Gi`
- storage: `20Gi`

### v1 model posture

`architect-juno` currently uses an **OpenRouter-only transition posture**.

This is deliberate and temporary.
It exists so the first operator-backed Architect plan can be materialized with the currently available provider path.
It should not be read as the permanent long-term Architect routing/provider design.

Current default model refs in the plan:

- primary: `openrouter/anthropic/claude-sonnet-4.6`
- fallback: `openrouter/openai/gpt-5.1-codex`

### What the plan means for users

When a user selects `spec.plan: architect-juno`, they are asking for:

- a non-toy Architect baseline
- enough CPU/memory/storage headroom for long-running interactive work
- a named, operator-curated default config profile rather than ad-hoc per-instance sizing

This first materialization intentionally stays narrow:

- it uses the existing operator plan mechanism
- it sets only supported resource/storage/config defaults
- it does **not** introduce a new operator API for richer Architect semantics yet

## Comparison hook for future plans

Future plans can be added to this catalog in the same format.

Suggested next comparison slot:

## `architect-ceres` *(reserved for future work)*

Use this section later to document how `architect-ceres` differs from `architect-juno`, for example in:

- target workload
- model posture
- resource envelope
- override policy
- session/runtime topology
