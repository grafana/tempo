---
title: Verify signed container images
menuTitle: Verify container images
description: Verify the cosign signature and SLSA build provenance of the container images Tempo publishes, before you deploy them.
weight: 650
---

# Verify signed container images

Starting with Tempo 3.1, Grafana Labs cryptographically signs every published Tempo container image and attaches a SLSA build provenance attestation to it.
Use these artifacts to confirm that an image actually came from the `grafana/tempo` repository and to trace it back to the exact source commit and CI run that built it, before you pull it into a cluster.

## Which images are signed

The following images published to [Docker Hub](https://hub.docker.com/u/grafana) are signed and attested:

- `grafana/tempo`
- `grafana/tempo-vulture`
- `grafana/tempo-query`
- `grafana/tempo-cli`

Signing and attestation run as a best-effort step after each image is published. They don't gate the release or block a deployment, so a signing failure never prevents an image from being available. It only means that image doesn't verify until it's re-signed.

## What you get

Each image ships with two supply chain artifacts:

- **A keyless cosign signature.** Proves the image was produced by the `sign-and-attest.yml` workflow in the `grafana/tempo` repository and hasn't been tampered with since. Grafana Labs doesn't hold or manage a private signing key: the workflow signs using a short-lived certificate issued by [Sigstore's](https://www.sigstore.dev/) Fulcio CA in exchange for a GitHub Actions OIDC token, and records the signing event in the public Rekor transparency log.
- **A SLSA build provenance attestation.** Binds the image's digest to the source commit, workflow, and run that built it. This is what lets you answer "was this image actually built by the Tempo CI pipeline from a specific commit, or did it come from somewhere else?"

Both are pushed as OCI referrers alongside the image, so they travel with it wherever the image is stored or mirrored.

## Before you begin

Install the verification tools:

- [cosign](https://docs.sigstore.dev/cosign/system_config/installation/) v2 or later, to verify the signature
- The [GitHub CLI](https://cli.github.com/) (`gh`), to verify the provenance attestation

## Verify an image

Pull the image by digest, or resolve the digest for a tag you already have, so you verify the exact artifact you're about to deploy rather than a tag that could later move to a different image:

```bash
docker buildx imagetools inspect grafana/tempo:<TEMPO_VERSION> --format '{{.Manifest.Digest}}'
```

### Verify the signature

Run `cosign verify` against the digest-pinned image, checking the certificate identity against the `sign-and-attest.yml` workflow in `grafana/tempo` and the GitHub Actions OIDC issuer:

```bash
cosign verify \
  --certificate-identity-regexp '^https://github\.com/grafana/tempo/\.github/workflows/sign-and-attest\.yml@' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  grafana/tempo@<DIGEST>
```

A successful verification prints the certificate details and a `Verified OK` message.

### Verify the provenance attestation

Run `gh attestation verify` against the same digest-pinned image, scoped to the `grafana/tempo` repository:

```bash
gh attestation verify oci://grafana/tempo@<DIGEST> --repo grafana/tempo
```

A successful verification confirms the attestation is signed by `grafana/tempo` and prints the predicate type and the workflow run that produced it.

## Result

If both commands succeed, you've confirmed that the image digest you're about to deploy was built and published by the Tempo CI pipeline from a known commit, and hasn't been altered since. If either command fails, don't deploy the image, and treat it as unverified until you can confirm why verification failed.

{{< admonition type="note" >}}
The verification identity (the workflow path plus the OIDC issuer) is a public contract. It stays the same across all four signed images, so you can reuse the same `--certificate-identity-regexp` and `--certificate-oidc-issuer` values regardless of which component you're verifying.
{{< /admonition >}}
