# `sbom-on-release.yml`

Reusable workflow that attaches a [Socket](https://socket.dev/)-generated
**SPDX software bill of materials (SBOM)** to a published Tempo GitHub release.
The SBOM lists the packages and versions that make up the release,
so security teams and OSS users can download it directly
instead of requesting one for each release.

The SBOM is uploaded as a release asset named
`<repo-name>-<tag>.spdx.json` (for example, `tempo-v3.1.0.spdx.json`).

> This is a different artifact, produced by a different workflow,
> than the container-image signature and SLSA provenance
> documented in [`sign-and-attest.md`](sign-and-attest.md).
> Those travel **with the image** as OCI referrers;
> this SBOM is attached **to the GitHub release** as a downloadable asset.

## Why it isn't triggered by `release: published`

`release-tag.yml` publishes the draft release
(created by `goreleaser`) with `secrets.GITHUB_TOKEN`,
and GitHub doesn't start a new workflow run for events produced by
the default `GITHUB_TOKEN`.
An `on: release: published` trigger would therefore never fire.
Instead, `release-tag.yml` calls this workflow explicitly
from its `export_sbom` job once the `publish` job has finished.

To (re)generate the SBOM for an existing release, run the workflow by hand with
`workflow_dispatch`.

## Inputs

Both `workflow_call` and `workflow_dispatch` take the same inputs:

| Input | Required | Default | Description |
| --- | --- | --- | --- |
| `version` | yes | — | The release tag to export an SBOM for, with or without a leading `v` (for example, `v3.1.0` or `v3.2.0-rc.1`). |
| `dry_run` | no | `false` | Generate the SBOM but do not upload it to the release. |

## Step flow

```
┌─────────────────────────────────────────────────────────────┐
│ export-sbom job                                              │
├─────────────────────────────────────────────────────────────┤
│ 1. Resolve tag and Socket branch                             │
│    - validate version is vX.Y.Z or vX.Y.Z-rc.N               │
│    - rc.N  ──▶ Socket branch "main"                          │
│    - X.Y.Z ──▶ Socket branch "release-vX.Y"                  │
│ 2. Checkout the exact release tag (Socket scans manifests    │
│    on disk, so the files must match the release)             │
│ 3. Get the Socket API token from Vault                       │
│ 4. Export SPDX SBOM from Socket (socket_org: grafana)        │
│ 5. Upload SBOM to the release (skipped when dry_run)         │
│    gh release upload "$TAG" "$SBOM_PATH" --clobber           │
└─────────────────────────────────────────────────────────────┘
```

The Socket branch is only the label Socket files the scan under.
The files actually scanned come from the tag checked out in step 2:
release candidates (`-rc.N`) are cut from `main`,
and minor and patch releases are cut from `release-vX.Y`
(see [`release-tag.yml`](release-tag.yml)).

## Permissions

The workflow declares `permissions: {}` at the top level
and grants the `export-sbom` job only what it needs:

- `contents: write` — to upload the SBOM as a release asset.
- `id-token: write` — to authenticate to Vault for the Socket API token.

## Behavior on failure

SBOM export runs after the release is already published,
so a failure here never blocks a release.
It only means that release has no SBOM asset until the workflow is re-run
with `workflow_dispatch`.
