# `sign-and-attest.yml`

Reusable workflow that adds cryptographic supply-chain guarantees to a published
container image. Given a **digest-pinned** image reference it produces, attaches,
and verifies two artifacts:

1. A **keyless cosign signature** — proves the image was produced by *this
   workflow's identity* and hasn't been tampered with.
2. A **SLSA build provenance attestation** — proves *how* and *from what* the
   image was built (image digest → source commit → workflow → run).

It is keyless throughout: there are **no signing keys or stored secrets**. Trust
comes from the workflow's GitHub Actions OIDC identity plus public transparency
logs.

> We use keyless cosign + SLSA provenance (Sigstore) because it is the
> foundation-backed, widely adopted standard — used by Kubernetes, npm, PyPI,
> and GitHub's own attestations — and it matches the approach already adopted by
> Grafana Alloy.

## Why

Image verification used to be manual (cross-referencing tag SHAs against the git
API), with no cryptographic chain from a CI run to a published image. This
workflow establishes that chain so anyone can verify an image is authentic and
trace it back to the exact commit and workflow run that built it.

## The three supply-chain artifacts

These are distinct and answer different questions:

| Artifact | Question it answers | Produced here? |
| --- | --- | --- |
| **Signature** (`cosign sign`) | Was this image produced by a trusted identity, untampered? | ✅ |
| **Provenance** (SLSA attestation) | How / from what was it built — which commit, workflow, run? | ✅ |
| **SBOM** | What packages/versions are inside it? | ❌ (possible follow-up) |

## How keyless signing works

There is no private key. The workflow gets a short-lived GitHub Actions OIDC
identity token; **Fulcio** (Sigstore's CA) exchanges it for a short-lived
certificate bound to that workflow identity; cosign signs the image *index
digest* with a freshly generated ephemeral key (discarded after); and **Rekor**
(Sigstore's public transparency log) records a timestamped entry. The Rekor
timestamp is what keeps the signature verifiable after the short-lived cert
expires — a verifier checks that it was signed *while* the cert was valid, not
that the cert is valid now. The SLSA provenance attestation is signed the same
keyless way and is pushed to the registry (as an OCI referrer) and the GitHub
attestations store.

## Inputs

| Input | Required | Default | Description |
| --- | --- | --- | --- |
| `image` | yes | — | Digest-pinned image ref, e.g. `us-docker.pkg.dev/.../tempo-vulture@sha256:...`. The workflow fails fast if it is not digest-pinned. |
| `registry` | no | `us-docker.pkg.dev` | Registry to authenticate against when writing the signature and attestation. |

## Step flow

```
┌─────────────────────────────────────────────────────────────┐
│ sign-and-attest job                                          │
├─────────────────────────────────────────────────────────────┤
│ 1. Parse & validate image                                    │
│    - require "@sha256:" (digest-pinned)                      │
│    - split into subject-name (repo) + subject-digest         │
│ 2. Login to GAR (OIDC workload identity)                     │
│ 3. Install cosign                                            │
│ 4. cosign sign      ──▶ keyless signature, logged to Rekor   │
│ 5. attest-build-provenance ──▶ signed SLSA provenance,       │
│                                pushed to registry + GH store │
│ 6. cosign verify    ──▶ confirm signature against identity   │
│ 7. gh attestation verify ──▶ confirm provenance vs repo      │
└─────────────────────────────────────────────────────────────┘
```

Steps 6–7 run in CI so a broken signature or attestation fails the run before
anyone relies on it. They check the **GAR** copy; the mirror to Docker Hub is
async and is not verified here.

## Verifying an image (for consumers)

Signature:

```bash
cosign verify \
  --certificate-identity-regexp '^https://github\.com/grafana/tempo/\.github/workflows/sign-and-attest\.yml@' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  us-docker.pkg.dev/grafanalabs-global/dockerhub-tempo-prod-mirror/tempo-vulture@sha256:...
```

Provenance:

```bash
gh attestation verify \
  oci://us-docker.pkg.dev/grafanalabs-global/dockerhub-tempo-prod-mirror/tempo-vulture@sha256:... \
  --repo grafana/tempo
```

> **Compatibility note:** the cosign verification identity
> (`.../sign-and-attest.yml@<ref>` + the OIDC issuer) is a public contract.
> Renaming or moving this workflow is a **breaking change** for anyone verifying.
