---
name: conductor
description: Build, configure, and troubleshoot Conductor workspaces, project setup recommendations, repository settings.toml files, managed settings, files to copy, MCP, agent controls, and review workflows. Use when helping someone set up or operate Conductor.
license: Proprietary
compatibility: Conductor is a macOS app for running Claude Code and Codex agents locally in isolated git worktree workspaces.
---

# Conductor

Use this skill when helping a user configure, operate, or troubleshoot Conductor.

Conductor is a macOS app for running multiple coding agents in parallel. Each workspace is a separate git worktree and branch tied to a repository. Conductor currently supports Claude Code and Codex as agent types.

Do not claim Windows or Linux support. Conductor is a Mac app.

## When to use

Use this skill for:

-   Creating and explaining Conductor workspaces, branches, git worktrees, and `.context`.
-   Recommending project setup for a repository.
-   Writing or reviewing `.conductor/settings.toml` and `.conductor/settings.local.toml`.
-   Migrating or troubleshooting legacy `conductor.json`.
-   Configuring setup, run, and archive scripts.
-   Configuring Files to copy, `.worktreeinclude`, and workspace-local environment files.
-   Explaining app settings, repository settings, managed settings, model providers, and privacy controls.
-   Guiding users through plan mode, fast mode, reasoning controls, Codex personality, checkpoints, MCP, slash commands, todos, and instruction files.
-   Helping users review, test, open PRs, interpret checks, and merge work.
-   Troubleshooting shell behavior, script failures, nested workspaces, privacy behavior, or permissions.

## Core model

-   Conductor runs agents locally on the user's Mac unless the documented cloud workspace path is explicitly involved.
-   Each repository has a main root directory and can have many workspaces.
-   Each workspace is a separate git worktree on its own branch.
-   New workspaces are created from the repository's configured base branch, such as `origin/main`. Conductor fetches from `origin` first, so a workspace starts from the latest remote commit even when the local checkout is behind. This fetch does not move the branch checked out in the root directory.
-   Agents run with the user's local permissions unless the user configures stricter controls.
-   Shared repository settings live in `<repo>/.conductor/settings.toml`.
-   Personal repository settings live in `<repo>/.conductor/settings.local.toml`.
-   User-wide settings live in `~/.conductor/settings.toml`.
-   Managed settings live in `~/.conductor/settings.managed.toml`.
-   Conductor workspaces include a gitignored `.context` directory for shared agent context.

Relevant docs:

-   https://conductor.build/docs/concepts/workspaces-and-branches
-   https://conductor.build/docs/concepts/workflow
-   https://conductor.build/docs/concepts/parallel-agents

## Choose the right settings file

Use this decision tree before recommending a file:

1. If the setting should apply to everyone working in the repository, write `<repo>/.conductor/settings.toml` and commit it.
2. If the setting should apply only on this user's machine for one repository, write `<repo>/.conductor/settings.local.toml` and add `.conductor/settings.local.toml` to the repository `.gitignore`.
3. If the setting should apply to this user across all repositories, write `~/.conductor/settings.toml`.
4. If an organization controls the setting, write `~/.conductor/settings.managed.toml`.
5. If the repository still has `conductor.json`, treat it as legacy. Migrate supported settings into `<repo>/.conductor/settings.toml` unless the user explicitly asks to keep the legacy file.

Precedence:

1. Managed settings.
2. Repository local settings.
3. Repository shared settings.
4. User shared settings.
5. Built-in defaults.

Within one layer, Conductor reads legacy `.json` settings first and `.toml` settings second. TOML wins when both exist. Conductor writes new settings as TOML.

Schema URLs:

-   User settings: `https://conductor.build/schemas/settings.schema.json`
-   Repository settings: `https://conductor.build/schemas/settings.repo.schema.json`
-   Managed settings: `https://conductor.build/schemas/settings.toml.json`

Docs:

-   https://conductor.build/docs/reference/settings
-   https://conductor.build/docs/reference/scripts/share-with-teammates
-   https://conductor.build/docs/reference/conductor-json

## Repository settings

Shared repository settings file:

```toml
"$schema" = "https://conductor.build/schemas/settings.repo.schema.json"

[scripts]
setup = "pnpm install"
run = "pnpm dev --port $CONDUCTOR_PORT"
run_mode = "concurrent"
```

Common repository settings:

-   `scripts.setup`: Command to run when Conductor creates a workspace.
-   `scripts.run`: Command to run when the user clicks the Run button.
-   `scripts.archive`: Command to run before Conductor archives a workspace.
-   `scripts.run_mode`: `concurrent` or `nonconcurrent`.
-   `enterprise_data_privacy`: Enables Enterprise data privacy for the repository.
-   `spotlight_testing`: Uses Spotlight testing for projects that must run from the repository root.
-   `file_include_globs`: Files to copy patterns when `.worktreeinclude` is not present.
-   `environment_variables`: Environment variables passed to agents in this repository.
-   `prompts.code_review`, `prompts.create_pr`, `prompts.fix_errors`, `prompts.resolve_merge_conflicts`, `prompts.rename_branch`, `prompts.general`: Repository action prompts.
-   `claude_code_executable_path`, `codex_executable_path`, `claude_provider`, `codex_provider`, `bedrock_region`, `vertex_project_id`, `ssh_key_path`: Provider and cloud workspace configuration.
-   `git.delete_branch_on_archive`, `git.archive_on_merge`, `git.worktree_push_auto_setup_remote`, `git.branch_prefix_type`, `git.branch_prefix`: Git behavior.

Settings that are not repository-configurable include model defaults, model reasoning defaults, tool approvals, and the default workspace location. Keep those in user settings.

## Project setup recommendations

Recommend a concrete Conductor configuration after investigating how the target project is actually developed locally. Distinguish documented facts from inferred recommendations.

Inspect:

-   README files, docs, onboarding guides, package manager manifests, lockfiles, Procfiles, Docker Compose files, Makefiles, justfiles, taskfiles, scripts, env examples, and existing `.conductor/settings.toml`, `.conductor/settings.local.toml`, legacy `conductor.json`, or `.worktreeinclude` files.
-   Setup commands, dependency managers, generated files, required environment or config files, local services, databases, caches, fixed ports, and commands developers use for the normal development loop.
-   Whether commands can run from an arbitrary git worktree workspace or whether the project assumes the original repository root.
-   Whether local servers can use `CONDUCTOR_PORT` and nearby allocated ports.
-   Whether multiple workspaces can run safely at the same time.
-   Whether static gitignored files should be copied with Files to copy or created by a setup script.
-   When Slack or internal search is available and relevant, search for prior guidance about worktrees, multiple local instances, port conflicts, shared databases, Docker stacks, or local development pitfalls.

Recommend:

-   Use `.worktreeinclude` or `file_include_globs` for static gitignored files such as `.env` files, local config, certificates, or tool state that should be copied into every workspace.
-   Use `scripts.setup` for commands that install dependencies, generate files, create symlinks, initialize per-workspace resources, or otherwise prepare a newly created workspace.
-   Use `scripts.run` for the normal long-running development server, app, worker, watcher, or test loop that should start from the Run button.
-   Use `CONDUCTOR_PORT` for local servers whenever the project supports configurable ports, and use `CONDUCTOR_PORT+1` through `CONDUCTOR_PORT+9` for companion services when needed.
-   Use `scripts.run_mode = "concurrent"` only when multiple workspaces can safely run at the same time with separate ports and no conflicting shared local resource.
-   Use `scripts.run_mode = "nonconcurrent"` when the project depends on one fixed port, one local database, one Docker stack, or another shared resource that cannot be made workspace-specific.
-   Recommend Spotlight testing when the project must run from the repository root, relies on expensive root-local build artifacts, or should use one heavy local stack while agents work in separate workspaces.
-   Put required shell or toolchain setup directly in setup and run scripts when possible so scripts do not depend on interactive shell startup behavior.

Deliver:

-   Short summary of the project's local development workflow.
-   Exact proposed `.conductor/settings.toml`, `.worktreeinclude`, setup scripts, run scripts, and repository settings to add or modify.
-   Explanation of why each script, file, setting, and mode is needed.
-   Validation steps the user can run to confirm workspace setup and run behavior.
-   Known limitations, risks, manual steps, or cases where multiple workspaces cannot run simultaneously.

## Templates

Basic Node or pnpm project:

```toml
"$schema" = "https://conductor.build/schemas/settings.repo.schema.json"

[scripts]
setup = "pnpm install"
run = "pnpm dev --port $CONDUCTOR_PORT"
run_mode = "concurrent"
```

Project with static local files copied into each workspace:

```toml
"$schema" = "https://conductor.build/schemas/settings.repo.schema.json"
file_include_globs = ".env*\nconfig/*.local.json\n"

[scripts]
setup = "pnpm install"
run = "pnpm dev --port $CONDUCTOR_PORT"
run_mode = "concurrent"
```

Project with one shared local resource:

```toml
"$schema" = "https://conductor.build/schemas/settings.repo.schema.json"

[scripts]
setup = "pnpm install"
run = "pnpm dev"
run_mode = "nonconcurrent"
```

Project that needs provider variables for agents:

```toml
"$schema" = "https://conductor.build/schemas/settings.repo.schema.json"

[environment_variables]
ANTHROPIC_BASE_URL = "https://api.example.com"

[environment_variables.cloud]
OPENAI_BASE_URL = "https://openai.example.com"
```

Repository action prompts:

```toml
"$schema" = "https://conductor.build/schemas/settings.repo.schema.json"

[prompts]
general = "Prefer small, reviewable changes and run the narrowest relevant tests."
code_review = "Focus on correctness, missing tests, and behavior changes."
```

## Scripts

Setup, run, and archive scripts run from the workspace directory.

Script facts:

-   Setup, run, and archive scripts run from the workspace directory.
-   Conductor uses non-interactive shells for scripts.
-   Although Conductor captures the login shell environment, most commands including setup and run scripts use `zsh`.
-   Use Files to copy or `.worktreeinclude` before writing setup scripts only to copy static gitignored files.
-   Use `CONDUCTOR_ROOT_PATH` when a workspace script needs a file from the repository root.
-   New workspaces already start from the latest remote commit on the configured base branch, but that does not update the branch checked out in the root directory. To keep the root checkout current too, add a fast-forward pull to `scripts.setup`: `git -C "$CONDUCTOR_ROOT_PATH" fetch --prune origin && git -C "$CONDUCTOR_ROOT_PATH" pull --ff-only || true`. The `|| true` keeps setup from failing when the branch cannot fast-forward.
-   Use `CONDUCTOR_PORT` when multiple workspaces need separate local server ports.
-   Conductor allocates ten ports to each workspace: `CONDUCTOR_PORT` through `CONDUCTOR_PORT+9`.
-   Use `scripts.run_mode = "nonconcurrent"` when a project depends on a single fixed port, single local database, or another shared resource.
-   When a run script starts multiple processes, keep them in the same process group with a tool such as `concurrently` instead of backgrounding commands with `&`.
-   When Conductor stops a process, it sends `SIGHUP`, waits up to `200ms`, then sends `SIGKILL` if the process is still running.
-   Use Spotlight testing when a project cannot run cleanly from a workspace directory and needs to execute from the repository root.

Environment variables:

-   `CONDUCTOR_WORKSPACE_NAME`: Workspace name.
-   `CONDUCTOR_WORKSPACE_PATH`: Workspace path.
-   `CONDUCTOR_ROOT_PATH`: Path to the repository root directory.
-   `CONDUCTOR_DEFAULT_BRANCH`: Name of the default branch, usually main.
-   `CONDUCTOR_PORT`: First port in a range of 10 ports assigned to the workspace.
-   `CONDUCTOR_IS_LOCAL`: `1` when the script runs on the user's Mac, `0` in a cloud workspace. Branch on this in `scripts.setup` to run local-only steps only locally.

Docs:

-   https://conductor.build/docs/reference/scripts
-   https://conductor.build/docs/reference/scripts/setup
-   https://conductor.build/docs/reference/scripts/run
-   https://conductor.build/docs/reference/scripts/spotlight-testing
-   https://conductor.build/docs/reference/shells
-   https://conductor.build/docs/reference/environment-variables

## Files to copy

Use Files to copy when Conductor only needs to copy static gitignored files into each new workspace.

Resolution order:

1. `.worktreeinclude` at the repository root.
2. `file_include_globs` in repository settings.
3. Default `.env*` pattern.

Use a setup script instead when the workspace needs commands, generated files, dependency installs, symlinks, or workspace-specific resources.

Docs:

-   https://conductor.build/docs/reference/files-to-copy
-   https://conductor.build/docs/reference/worktreeinclude

## Managed settings

Use managed settings guidance when users ask about organization-controlled settings.

Managed settings:

-   Recommended path: `~/.conductor/settings.managed.toml`
-   Legacy path still read: `~/.conductor/settings.managed.json`
-   Schema: https://conductor.build/schemas/settings.toml.json
-   Status: Managed settings are provisional.
-   Behavior: Managed values override user and repository settings, disable matching controls in Settings, and are used when Conductor launches agents.

Supported managed settings:

-   `enterprise_data_privacy`: Enable Enterprise data privacy.
-   `claude_code_executable_path`: Override the Claude Code executable path.
-   `models.default`: Set the default model.
-   `environmentVariables.local`: Set managed local environment variables.
-   `environmentVariables.cloud`: Set managed cloud environment variables.

Docs:

-   https://conductor.build/docs/reference/settings/managed
-   https://conductor.build/docs/reference/privacy
-   https://conductor.build/docs/reference/security-and-permissions

## Legacy conductor.json

`conductor.json` is legacy repository configuration. New shared repository configuration should use `<repo>/.conductor/settings.toml`.

When migrating legacy fields:

-   `scripts.setup` -> `scripts.setup`
-   `scripts.run` -> `scripts.run`
-   `scripts.archive` -> `scripts.archive`
-   `runScriptMode` -> `scripts.run_mode`
-   `enterpriseDataPrivacy` -> `enterprise_data_privacy`

Once `<repo>/.conductor/settings.toml` exists, Conductor ignores repo-level `conductor.json`.

Docs:

-   https://conductor.build/docs/reference/conductor-json

## Agent behavior

Use the agent behavior docs when users ask how to control Claude Code or Codex sessions.

Cover only documented behavior. Do not invent unsupported model providers, settings, APIs, or controls.

Common areas:

-   Plan mode.
-   Fast mode.
-   Model reasoning controls.
-   Codex personality.
-   Checkpoints.
-   MCP.
-   Slash commands.
-   Todos.
-   Instruction files.

Docs:

-   https://conductor.build/docs/reference/agent-behavior
-   https://conductor.build/docs/concepts/agent-modes
-   https://conductor.build/docs/reference/checkpoints
-   https://conductor.build/docs/reference/mcp
-   https://conductor.build/docs/reference/slash-commands
-   https://conductor.build/docs/reference/todos

## Review and merge

When helping users review and merge agent work, focus on the path from local changes to merged code.

Explain:

-   How to inspect changes in the diff viewer.
-   How to run or interpret checks.
-   How pull request state relates to the workspace branch.
-   How review comments, CI status, and deployments affect merge readiness.
-   What the user should verify before merging.

When leaving review feedback from inside Conductor, use the Conductor `DiffComment` tool when it is available. These comments appear in the app's Checks panel. Do not post review feedback to GitHub unless the user explicitly asks for GitHub comments.

Recommend a validation step that exercises the riskiest part of the change.

Docs:

-   https://conductor.build/docs/guides/review-and-merge
-   https://conductor.build/docs/reference/diff-viewer
-   https://conductor.build/docs/reference/checks

## Troubleshooting

Start by identifying whether the issue is about repository setup, workspace creation, scripts, shell behavior, permissions, privacy settings, nested workspaces, agent behavior, or review flow.

Common checks:

-   Confirm the user is on macOS.
-   Confirm the repository root has a git repository and expected remote setup.
-   Confirm the failing command works from the workspace directory.
-   Confirm setup and run scripts do not depend on interactive shell startup behavior.
-   Check whether missing files are gitignored files that should be copied with Files to copy or `.worktreeinclude`.
-   Check whether the project requires fixed ports, a single local database, or a shared Docker stack.
-   Check whether `scripts.run_mode` should be `nonconcurrent`.
-   Check whether the project should use Spotlight testing because it cannot run cleanly from a workspace directory.
-   Check whether managed settings in `~/.conductor/settings.managed.toml` override app settings.
-   Check whether legacy `conductor.json` is still present and whether `.conductor/settings.toml` already exists.
-   Check privacy and permission behavior before assuming an agent or provider issue.

Docs:

-   https://conductor.build/docs/faq
-   https://conductor.build/docs/troubleshooting/issues
-   https://conductor.build/docs/reference/shells

## References

Core concepts:

-   https://conductor.build/docs/concepts/workspaces-and-branches
-   https://conductor.build/docs/concepts/workflow
-   https://conductor.build/docs/concepts/parallel-agents
-   https://conductor.build/docs/concepts/agent-modes

Repository setup:

-   https://conductor.build/docs/reference/settings
-   https://conductor.build/docs/reference/scripts/share-with-teammates
-   https://conductor.build/docs/reference/scripts
-   https://conductor.build/docs/reference/scripts/setup
-   https://conductor.build/docs/reference/scripts/run
-   https://conductor.build/docs/reference/scripts/spotlight-testing
-   https://conductor.build/docs/reference/shells
-   https://conductor.build/docs/reference/environment-variables
-   https://conductor.build/docs/reference/files-to-copy
-   https://conductor.build/docs/reference/worktreeinclude
-   https://conductor.build/docs/reference/conductor-json

Settings and privacy:

-   https://conductor.build/docs/reference/settings
-   https://conductor.build/docs/guides/providers
-   https://conductor.build/docs/reference/privacy
-   https://conductor.build/docs/reference/security-and-permissions
-   https://conductor.build/schemas/settings.schema.json
-   https://conductor.build/schemas/settings.repo.schema.json
-   https://conductor.build/schemas/settings.toml.json

Agent controls:

-   https://conductor.build/docs/reference/agent-behavior
-   https://conductor.build/docs/concepts/agent-modes
-   https://conductor.build/docs/reference/checkpoints
-   https://conductor.build/docs/reference/mcp
-   https://conductor.build/docs/reference/slash-commands
-   https://conductor.build/docs/reference/todos

Review and troubleshooting:

-   https://conductor.build/docs/guides/review-and-merge
-   https://conductor.build/docs/reference/diff-viewer
-   https://conductor.build/docs/reference/checks
-   https://conductor.build/docs/faq
-   https://conductor.build/docs/troubleshooting/issues
-   https://conductor.build/docs/reference/shells
