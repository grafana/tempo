---
title: Recommended versions
menuTitle: Recommended versions
description: Learn about the Grafana Tempo version recommendation policy, including which versions Grafana recommends and which have reached end of life.
weight: 520
---

<!-- vale Grafana.We = NO -->
<!-- vale Grafana.Will = NO -->
<!-- vale Grafana.Timeless = NO -->

# Recommended versions

Grafana Tempo follows a regular release cadence.
Knowing which versions Grafana recommends helps you plan upgrades and make sure you receive bug fixes and security patches.

This page applies to self-managed Tempo installations.
Grafana Cloud follows a different release process.

## Release types

Tempo releases use semantic-like versioning (`MAJOR.MINOR.PATCH`):

- Major release (roughly once a year): May include new architecture, significant features, and breaking changes that require migration steps.
- Minor release (roughly every three to four months): Includes new features and enhancements. May occasionally include minor breaking changes documented in the [upgrade guide](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/).
- Patch release (as needed): Includes bug fixes and security patches only. No new features.

## Version recommendation policy

These timelines are guidance; always consult the release notes for the most up-to-date information.
For self-managed Tempo, version recommendations follow these rules:

- Each minor release receives bug fixes and security patches for **9 months** after its release date.
- The last minor release of a major version receives bug fixes and security patches for **15 months** after its release date.

Recommendation levels change as new versions are released:

| Status | What it means |
|---|---|
| **Recommended** | The current major version. It receives new minor releases with features approximately every three to four months, and all minor versions within the major receive patch releases until end of life. |
| **Maintained** | The minor version receives patch releases (bug fixes and security patches) until end of life. No new features. |
| **End of life (EOL)** | The version is past its patch window and receives no updates. Upgrade to a recommended version. |

High severity security vulnerabilities and critical feature degradation incidents may result in ad-hoc patch releases outside the normal schedule.

## Current versions

Here is an overview of version status:

| Version | Release date | Patch end date | Status |
|---|---|---|---|
| 3.0.x | May 28, 2026 | February 28, 2027 | Recommended |
| 2.10.x (last minor of 2.x) | January 26, 2026 | April 26, 2027 | Maintained |
| 2.9.x | October 8, 2025 | December 31, 2026 | Maintained |
| 2.8.x | June 10, 2025 | March 10, 2026 | EOL |
| 2.7.x | January 13, 2025 | October 13, 2025 | EOL |

{{< admonition type="warning" >}}
Tempo 2.8 has reached end of life (EOL) and no longer receives bug fixes or security patches.
If you're running Tempo 2.8, upgrade to a [recommended version](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/).
{{< /admonition >}}

## Upgrade strategies

Based on your needs, choose an upgrade cadence:

| Strategy | Advantages | Considerations |
|---|---|---|
| **Minor** (upgrade every release) | Small changelog to review. Access to latest features. Highest compatibility with other Grafana products. | Requires more frequent upgrades. |
| **Major** (upgrade once a year) | Yearly upgrade aligned with major releases. | Large changelog to review. May require migration steps for breaking changes. |

For the best experience, follow the minor release cadence. You can occasionally extend to a full quarter and still receive security fixes for your currently deployed version.

Before upgrading, always read the [upgrade guide](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/) for your target version.

## Minimize upgrade risk

- Test first: Test upgrades in a non-production environment before deploying to production.
- Back up: Back up your configuration and data before upgrading.
- Stay current: Running a recommended version means you receive security patches and bug fixes.
- Read the changelog: Review the [release notes](/docs/tempo/<TEMPO_VERSION>/release-notes/) and [upgrade guide](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/) before upgrading.
