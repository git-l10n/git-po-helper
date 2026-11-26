# AGENTS.md - GitHub Actions Maintenance Guide

## Scope

These instructions apply to files under `.github/` and to related CI
scripts when modifying workflows. Follow the repository-wide development
and commit conventions in the root `AGENTS.md`.

Preserve the supported Go versions, Linux and macOS coverage, and existing
build and test steps unless the requested change explicitly affects them.

## Action Version Updates

When updating GitHub Actions dependencies:

1. Inventory all workflow files and their `uses:` references, including
   actions and reusable workflows.
2. Check upstream stable releases, migration guides, and the relevant
   `action.yml` for supported inputs, outputs, and changed defaults.
3. Verify runner and action runtime requirements (such as Node.js),
   platform support, and compatibility with the Go version matrix.
4. Select the latest compatible stable release and update matching action
   references consistently across jobs and workflows. Explain any version
   held back because of compatibility concerns.
5. Review the resulting diff and run the applicable validation checks below.

Use major-version tags for GitHub Actions (for example,
`actions/checkout@vN`), not commit SHA references. This is a deliberate
project policy favoring maintainability, automatic compatible updates,
and avoiding dependence on the underlying Git object hash format as Git
evolves toward SHA-256. GitHub recommends immutable references for
stronger supply-chain guarantees; tags can move, and this project accepts
that tradeoff. Do not replace tags with commit SHAs solely to satisfy a
security scanner. Reassess the trustworthiness of action sources during
each update, and prefer official or actively maintained actions.

## Security Guidelines

- Grant `GITHUB_TOKEN` only the permissions required by the workflow.
  For ordinary CI, default to `permissions: { contents: read }`.
  Scope any necessary additional permissions to the individual job.
- Set `persist-credentials: false` for checkout when subsequent steps
  do not need authenticated Git operations.
- Prefer `push` and `pull_request` for ordinary CI. Do not run untrusted
  pull request code in privileged `pull_request_target` or `workflow_run`
  contexts.
- Do not interpolate untrusted event data directly into inline Shell
  commands. Pass it through environment variables and quote expansions.
- Do not hardcode credentials or expose tokens, secrets, or sensitive
  values in logs.
- Keep Go dependency caching disabled in the current CI workflow. Before
  enabling any cache, assess the performance benefit, cache scope, trust
  boundaries, and cache poisoning risks.
- Review security findings in context. Distinguish GitHub recommendations
  from project-specific policy and report intentional exceptions.

## Workflow Design

- Keep workflow YAML focused on orchestration. Reuse existing scripts
  and Makefile targets rather than duplicating their implementation.
- Call `test/install-prereq.sh` without `sudo`. Elevate only individual
  package manager commands within the script when necessary. Run Homebrew
  as a normal user and use `GITHUB_PATH` for tools needed by later steps.
- Specify Bash explicitly for workflow steps to obtain consistent error
  and pipeline failure handling. Keep portable shared scripts compatible
  with their declared interpreter.
- Set reasonable job timeouts and cancel superseded CI runs. Ensure the
  concurrency group distinguishes workflows, event types, and refs.
- Pin the static analysis tool to a specific released version instead of
  using `@latest`. Verify its Go toolchain compatibility when updating it.
- Run static analysis after build and test steps so their results remain
  available if installing the analysis tool fails.
- Preserve `fetch-depth: 0` where the tests require complete Git history.
  Do not reduce the existing Go version matrix or test coverage merely
  to simplify a dependency upgrade.

## Validation

Run the applicable checks from the repository root after modifying a
workflow. Use current compatible versions of the validation tools.

```sh
actionlint
shellcheck test/install-prereq.sh
sh -n test/install-prereq.sh
zizmor --offline --persona pedantic .github/workflows
git diff --check
make lint
make ut
```

- Ensure ShellCheck is available to actionlint for inline Shell analysis.
  Check shared scripts separately when they are changed, including their
  platform-specific installation behavior where applicable.
- Resolve syntax errors and unintended security findings. Report the
  known zizmor `unpinned-uses` findings for major-version action tags as
  an intentional project policy difference; do not suppress them merely
  to claim a clean audit.
- Offline auditing does not check findings that require network access.
  Consult upstream release and security information when updating actions.
- Follow the root `AGENTS.md` for additional project checks, including
  integration tests before promoting a version.
- Local checks do not replace actual workflow runs on GitHub-hosted
  runners. When a CI run is available, verify both platforms and all
  matrix entries, including static analysis and integration tests.
- Report which checks ran, their outcomes, known exceptions, and any
  checks that could not be run. Do not claim unverified CI success.

## Commit Guidelines

Follow the root `AGENTS.md` for English Conventional Commits, commit
message structure, and pre-commit checks. Group changes by independent
maintenance concerns so they can be reviewed and reverted separately.

For example:

```text
ci: upgrade GitHub Actions to vN
ci: harden workflow security
ci: limit workflow execution time and concurrency
ci: explicitly use Bash for workflow steps
ci: pin staticcheck version
ci: limit privileges for prerequisite installation
```

Keep a shared script change and the corresponding workflow call-site
changes in the same atomic commit. Do not mix unrelated Go code changes
with workflow maintenance.

## References

- [GitHub Actions workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax)
- [GitHub Actions secure use reference](https://docs.github.com/en/actions/reference/security/secure-use)
- [GitHub Changelog](https://github.blog/changelog/)
- [actions/checkout releases](https://github.com/actions/checkout/releases)
- [actions/setup-go releases](https://github.com/actions/setup-go/releases)
- [actionlint](https://github.com/rhysd/actionlint)
- [ShellCheck](https://github.com/koalaman/shellcheck)
- [zizmor](https://docs.zizmor.sh/)
- [Staticcheck](https://staticcheck.dev/)
