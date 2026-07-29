# Changelog entries

Tempo manages `CHANGELOG.md` with [chloggen](../tools/chloggen) (the in-tree tool
at `tools/chloggen`). Instead of editing `CHANGELOG.md` directly, every PR adds a
small YAML file to this directory; the files are collated into the changelog at
release time. This avoids merge conflicts on a shared file.

## Add an entry

```bash
make chlog-new                     # creates .chloggen/<your-branch>.yaml from TEMPLATE.yaml
make chlog-new FILENAME=my-change  # optional: name the file explicitly
```

The filename defaults to the current branch name; pass `FILENAME=` to override
it (required on `main`/`master` or a detached HEAD). Edit the generated file:

```yaml
change_type: enhancement       # breaking | change | feature | enhancement | bug_fix | security
component: metrics-generator   # must be in the components allowlist in config.yaml
note: A brief description of the change.
issues: []                     # (optional) PR number(s); blank = auto-filled at release
subtext:                       # optional extra detail
user: your-github-handle       # rendered as "(@your-github-handle)"
```

`issues` is optional. If you leave it blank, `chlog-update` fills it at release
time with the PR number parsed from the commit that added this file (Tempo
squash-merges PRs as `... (#1234)`). On a release branch the file is added by the
backport PR, so the backport PR number is used; to pin a specific number instead,
set `issues` explicitly.

If the PR cannot be determined (for example the entry was committed directly to
`main`, or its PR is not yet merged), both `chlog-update` and `chlog-preview`
fail and list the entries needing an explicit `issues` — they never render an
entry without a PR link.

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

## Writing the note

Keep the note brief — one or two sentences at most —
and focus on user impact.
Internals of how the change is made are rarely relevant;
when they are, the note should still open with the user impact,
and extra implementation detail belongs in `subtext`, not the note.

For example, for a change that adds predicate pushdown to the parquet iterators:

- ✅ `Improve read performance by pushing down predicates to the parquet iterators.`
- ❌ `Add support for pushdown predicates in the parquet iterators.`

The note is about what users get, not the technical improvement itself.

## Validate / preview

```bash
make chlog-validate   # validate all entry files
make chlog-preview    # render the pending entries to stdout
```

## Release (maintainers)

```bash
make chlog-update VERSION=v2.11.0   # collate entries into CHANGELOG.md and delete them
```
