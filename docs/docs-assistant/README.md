# Docs Assistant v1.0.0

Automatically update user documentation when pull requests change user-facing behaviour.

An AI agent reviews PR diffs against a `context.yaml` coverage plan and updates `docs/sources/` when user-facing changes are detected. A separate workflow checks for broken Markdown links on every PR.

> Designed for Grafana Hugo repos where docs live alongside product code. Not a general-purpose docs agent — quality depends on a well-maintained `context.yaml`.

## Author

[Sean Packham](https://github.com/grafsean), Principal Technical Writer at Grafana Labs.

Want a custom agent tailored to your repo? Reach out to Sean in the #docs Slack channel.

## Setup

### 1. Install files

```bash
bash scripts/install-docs-assistant.bash /path/to/your/repo
```

Copies all files into `docs/docs-assistant/` and places both workflows in `.github/workflows/`.

### 2. Customise the writing guide

Edit `docs/docs-assistant/prompts/docs-assistant.md` with your product name, terminology, style conventions, and article templates. This is the main file to tailor per product.

### 3. Generate context.yaml

```bash
cat docs/docs-assistant/prompts/generate-context.md | claude
```

Review and adjust. Aim for 2–4 coverage points per article — enough for the agent to detect relevant changes, not so much that context becomes noisy.

### 4. Configure the Anthropic API key

Edit the "Get Anthropic secret" step in `.github/workflows/docs-assistant-update.yaml` to match your secret management approach:

- **GitHub secret:** replace the vault step with `echo "ANTHROPIC_API_KEY=${{ secrets.ANTHROPIC_API_KEY }}" >> "$GITHUB_ENV"`
- **Vault / OIDC:** keep or adapt the existing step

### 5. Check links locally

**Tempo docs (Grafana Hugo):** run with `--grafana-hugo` so `/docs/…`, `/media/…`, `ref:`, and other site paths are not reported as errors. Without this flag, the checker assumes relative-only links and will produce many false positives.

```bash
python3 docs/docs-assistant/links.py --grafana-hugo docs/sources/
# or a single section:
python3 docs/docs-assistant/links.py --grafana-hugo docs/sources/tempo/configuration
```

`run-local-checks.sh` passes `--grafana-hugo` by default. Use `LINK_CHECK_STRICT=1` only if you want the original strict behavior.

**Cron / launchd and log files:** see [`.agents/doc-agents/README-LOCAL-CHECKS.md`](../../.agents/doc-agents/README-LOCAL-CHECKS.md).

For repos that use only relative in-repo links, the flag is optional:

```bash
python3 docs/docs-assistant/links.py docs/sources/
```

Fix any errors before enabling the workflow. The checker catches `.md` extensions and missing targets; in strict mode it also flags absolute `/docs/` paths.

## Usage

### Update docs

Trigger on any pull request:

| Method      | How                                   |
| ----------- | ------------------------------------- |
| **Comment** | Post `/docs-update` on a pull request |
| **Label**   | Add the `docs-update` label           |

The agent reviews the PR diff, updates docs if needed, commits to the PR branch, and posts a summary comment. A `docs-assistant` label is added to every PR the agent processes.

### Link checking

Runs automatically on PRs that touch `docs/sources/`. Posts a comment and fails the check if broken links are found.

## Notes

- **Manual trigger only.** The update agent runs only when explicitly requested — no unexpected CI spend.
- **Always review.** Treat agent commits as a first draft.
- **Not 100% coverage.** Works well for scoped, incremental changes. Large rewrites still need human authoring.
- **Fork PRs.** Docs writeback requires a same-repo branch. The agent skips fork PRs and posts a comment.

## GitHub token permissions

The `docs-assistant-update` workflow needs these permissions on `GITHUB_TOKEN`:

| Permission             | Why                                        |
| ---------------------- | ------------------------------------------ |
| `contents: write`      | Push docs commits to the PR branch         |
| `pull-requests: write` | Post summary comments and add labels       |
| `issues: write`        | Post comments on `issue_comment` events    |
| `id-token: write`      | Only if using OIDC-based secret management |

## Files

After install, files are placed at:

| File                                              | Purpose                                      |
| ------------------------------------------------- | -------------------------------------------- |
| `docs/docs-assistant/prompts/docs-assistant.md`   | Writing guide and style rules for the agent  |
| `docs/docs-assistant/prompts/update-docs.md`      | Agent prompt — edit to change behaviour      |
| `docs/docs-assistant/prompts/generate-context.md` | Prompt for generating `context.yaml` locally |
| `docs/docs-assistant/context.yaml`                | Coverage plan — what each article must cover |
| `docs/docs-assistant/links.py`                    | Link checker script                          |
| `.github/workflows/docs-assistant-update.yaml`    | Docs update workflow                         |
| `.github/workflows/docs-assistant-links.yaml`     | Link checker workflow                        |

## Development

Prerequisites: [actionlint](https://github.com/rhysd/actionlint).

```bash
brew install actionlint
npm install
```

| Command          | What it does                        |
| ---------------- | ----------------------------------- |
| `npm run lint`   | Lint workflow files with actionlint |
| `npm run format` | Format all files with Prettier      |
| `npm test`       | Run Python unit tests               |

The pre-commit hook runs `npm run format` (Prettier) and `npm run lint` (actionlint) automatically before every commit. Run `npm install` once to activate it.
