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

There is no private key. At signing time:

1. **OIDC token** — the workflow gets a short-lived GitHub Actions OIDC identity
   token (`https://github.com/grafana/tempo/.github/workflows/sign-and-attest.yml@<ref>`).
2. **Fulcio** (Sigstore's CA) exchanges that OIDC token for a **short-lived
   signing certificate** (~10 min) binding a freshly generated ephemeral key to
   that workflow identity. The key is discarded after signing.
3. **cosign** signs the image *index digest* with the ephemeral key.
4. **Rekor** (Sigstore's public, append-only transparency log) records a
   timestamped entry for the signature.

The Rekor timestamp is what makes a signature from a short-lived cert verifiable
forever: a verifier checks "was this signed while the cert was valid?" (proved by
the log entry) rather than "is the cert valid right now?" (it expired minutes
after signing).

```
GitHub OIDC identity ──▶ Fulcio short-lived cert ──▶ ephemeral key signs digest
                                                              │
                                                  logged + timestamped in Rekor
                                                              │
                                    verifiable forever — no key/cert to store or rotate
```

The SLSA provenance attestation is signed the same keyless way and is both
pushed to the registry (as an OCI referrer next to the image) and recorded in the
GitHub attestations store.

## Inputs

| Input | Required | Default | Description |
| --- | --- | --- | --- |
| `image` | yes | — | Digest-pinned image ref, e.g. `us-docker.pkg.dev/.../tempo-vulture@sha256:...`. The workflow fails fast if it is not digest-pinned. |
| `registry` | no | `us-docker.pkg.dev` | Registry to authenticate against when writing the signature and attestation. |

## Permissions

| Permission | Why |
| --- | --- |
| `contents: read` | Checkout / repo context. |
| `id-token: write` | Mints the OIDC token for keyless cosign, provenance signing, and GAR workload-identity login. |
| `attestations: write` | Writes the SLSA provenance attestation to the GitHub attestations store. |

No `secrets: inherit` — everything is OIDC.

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

## How it's wired into the docker build (`docker.yml`)

The `docker` and `manifest` jobs build per-arch images and push a multi-arch
manifest under
`us-docker.pkg.dev/grafanalabs-global/dockerhub-tempo-prod-mirror/<component>`.
Two jobs then drive signing for `tempo-vulture`:

1. `resolve-vulture-digest` — resolves the pushed `:$TAG` manifest to its
   immutable **index digest** (`docker buildx imagetools inspect`) and outputs a
   digest-pinned ref. Signing by digest, not by mutable tag, binds the signature
   to exact bytes.
2. `sign-vulture` — calls this reusable workflow with that digest-pinned ref.

```
docker (per-arch build+push) ─▶ manifest (multi-arch push)
                                      │
                                      ▼
                         resolve-vulture-digest  (tag ─▶ @sha256 digest)
                                      │
                                      ▼
                               sign-vulture  ─uses─▶ sign-and-attest.yml
```

Signing/attestation runs **after** publish and is intentionally **not a gate** on
the release or `cd-to-dev-env`: a pushed image is immutable and can't be
re-signed, so it is signed best-effort. A failure surfaces as a failed check on
the run; alerting on `sign-vulture` failures is recommended.

## Scope

Currently scoped to **`tempo-vulture`** to prove out the flow. The other
components (`tempo`, `tempo-query`, `tempo-cli`) can be added by widening the
resolve step into the existing matrix and calling this workflow per component.

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

## Why this approach is solid

The keyless flow relies on **Sigstore** (Fulcio + Rekor). It is a well-established,
foundation-governed standard rather than a single-vendor tool:

- **Governance & backing.** Sigstore is an open-source project under the
  [OpenSSF](https://openssf.org/) / Linux Foundation, founded and maintained by
  Google, Red Hat, and Purdue University with a broad community of maintainers.
  The public Fulcio/Rekor instances are operated as a community good and the
  project has undergone independent third-party security audits.
- **Generally available & stable.** Sigstore has been GA since 2022 with a public
  API stability guarantee, and `cosign` is the de-facto standard for container
  signing.
- **Load-bearing for major ecosystems.** The exact Fulcio + Rekor flow used here
  underpins:
  - **Kubernetes** — signs all release images and artifacts.
  - **npm** and **PyPI** — package provenance / attestations.
  - **GitHub** — `actions/attest-build-provenance` (used by this workflow) is
    built on Sigstore.
  - **OpenTelemetry Collector**, **Chainguard**, distroless images, and many
    CNCF projects.
- **Consistent with Grafana.** Mirrors the keyless cosign approach already adopted
  by Grafana Alloy.

### Trust trade-offs to be aware of

- Signing has a **runtime dependency on the public Sigstore instances**
  (`fulcio.sigstore.dev` / `rekor.sigstore.dev`). An outage breaks *signing*, not
  verification of already-logged signatures. ([status page](https://status.sigstore.dev))
- **Rekor is a public log** — everything written is world-readable forever. That
  is fine for image digests and workflow identities, which are not secret.
- Verification strength depends on **pinning the exact identity** (the
  `--certificate-identity-regexp` + `--certificate-oidc-issuer` above). A loose
  identity match weakens the guarantee.

### References

- Sigstore overview — https://docs.sigstore.dev/about/overview/
- Fulcio (the CA) — https://docs.sigstore.dev/certificate_authority/overview/
- Rekor (the transparency log) — https://docs.sigstore.dev/logging/overview/
- cosign verifying — https://docs.sigstore.dev/cosign/verifying/verify/
- SLSA provenance — https://slsa.dev/
