# Changelog entries

Tempo manages `CHANGELOG.md` with [chloggen](../tools/chloggen) (the in-tree tool
at `tools/chloggen`). Instead of editing `CHANGELOG.md` directly, every PR adds a
small YAML file to this directory; the files are collated into the changelog at
release time. This avoids merge conflicts on a shared file.

## Add an entry

```bash
make chlog-new        # creates .chloggen/<your-branch>.yaml from TEMPLATE.yaml
```

Edit the generated file:

```yaml
change_type: enhancement       # breaking | change | feature | enhancement | bug_fix | security
component: metrics-generator   # must be in the components allowlist in config.yaml
note: A brief description of the change.
issues: [1234]                 # PR or issue number(s)
subtext:                       # optional extra detail
user: your-github-handle       # rendered as "(@your-github-handle)"
```

`change_type` keywords render under these sections:

| keyword       | section               |
| ------------- | --------------------- |
| `breaking`    | 🛑 Breaking changes 🛑 |
| `change`      | 🔧 Changes 🔧          |
| `feature`     | 🚀 Features 🚀         |
| `enhancement` | 💡 Enhancements 💡     |
| `bug_fix`     | 🧰 Bug fixes 🧰        |
| `security`    | 🔒 Security 🔒         |

`component` must be one of the values in the `components` allowlist in
[`config.yaml`](./config.yaml); `chlog-validate` rejects anything else. Adding a
new component? Add it to that list in the same PR.

## Validate / preview

```bash
make chlog-validate   # validate all entry files
make chlog-preview    # render the pending entries to stdout
```

## Release (maintainers)

```bash
make chlog-update VERSION=v2.11.0   # collate entries into CHANGELOG.md and delete them
```
