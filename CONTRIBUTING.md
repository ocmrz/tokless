# Contributing to tokless

tokless is a thin orchestrator: it installs token-saving tools and wires them into AI coding agents. The commands (`init`, `update`, `doctor`, `uninstall`) are **pure iteration** over a registry of agents and tools — they never switch on a specific agent or tool. So adding either is local and small.

| Add a new… | Files |
| ---------- | ----- |
| **Agent** | `src/agents/<name>.ts` + one line in `src/agents/index.ts` + the id in `src/core/ids.ts` |
| **Tool** | `src/tools/<name>.ts` + one line in `src/tools/index.ts` + the id in `src/core/ids.ts` |

You never edit a command file, another agent, or another tool.

## Layout

```
src/
├── core/         ids.ts (union types) · agent-manifest.ts · tool-manifest.ts · registry.ts
├── agents/       one file per agent (export default AgentManifest) + index.ts registration
├── tools/        one file per tool (export default ToolManifest) + index.ts registration
├── commands/     init · update · doctor · disable/uninstall — iterate the registries
├── util/         exec, paths, json, toml, mcp-spawn, npm-install, prompt, …
└── bootstrap.ts  imports agents/index + tools/index; each command imports this once
```

## Adding a tool

Write `src/tools/<name>.ts` exporting a `ToolManifest`:

```ts
import { which } from "../util/exec.js";
import { npmGlobalInstall } from "../util/npm-install.js";
import type { ToolManifest, RunOpts } from "../core/tool-manifest.js";

async function install(opts: RunOpts): Promise<boolean> {
  if (which("mytool") && !opts.upgrade) return true;   // skip if present, unless `update`
  if (opts.dryRun) return true;
  return (await npmGlobalInstall("mytool", "latest")) !== null;  // returns version | null
}

const mytool: ToolManifest = {
  id: "mytool",
  label: "My Tool",
  description: "One line.",
  homepage: "https://github.com/owner/mytool",
  channel: "npm",            // "npm" | "github" | "manual"
  install,
  wireFor: {                 // one entry per agent you support; omit an agent to skip it
    claude:   async (opts) => { /* merge entry into ~/.claude.json */ return true; },
    opencode: async (opts) => { /* merge entry into opencode.json */ return true; },
    codex:    async (opts) => { /* merge entry into ~/.codex/config.toml */ return true; },
  },
  unwireFor?: { /* optional — for `uninstall`/`disable` */ },
  verifyFor?: { /* optional — for `doctor` */ },
};
export default mytool;
```

Register it in `src/tools/index.ts` (`registerTool(mytool)`) and add `"mytool"` to `ToolId` in `src/core/ids.ts`.

## Adding an agent

Write `src/agents/<name>.ts` exporting an `AgentManifest`:

```ts
import * as fs from "node:fs";
import { agentPaths } from "../util/paths.js";
import { which } from "../util/exec.js";
import type { AgentManifest } from "../core/agent-manifest.js";

const cursor: AgentManifest = {
  id: "cursor",
  label: "Cursor",
  homepage: "https://cursor.com",
  cliBin: "cursor",
  configDir: () => agentPaths.cursor().dir,
  detect: () =>
    which("cursor") || fs.existsSync(agentPaths.cursor().dir)
      ? { installed: true, source: which("cursor") ? "cli" : "config" }
      : { installed: false, source: null },
};
export default cursor;
```

Register it in `src/agents/index.ts`, add the id to `AgentId` in `src/core/ids.ts`, and add a `cursor()` entry to `agentPaths` in `src/util/paths.ts`. Existing tools opt in by adding a `cursor` key to their `wireFor` map — tools that don't are skipped for that agent, no error.

## Discipline

- **Cite the upstream docs** for every config shape you write, in your PR. Quote the exact section. No invented config keys.
- **Use the tool's own installer/CLI** where it has one; only author config files by hand when it doesn't.
- **Bare binaries over `npx`** for runtime spawn — use `pickMcpSpawn(bin)` from `util/mcp-spawn.ts`. `tokless update` is the only path that bumps versions.
- **Merge, never overwrite.** Read the existing file, insert only your keys, write back.
- **Idempotent.** Running tokless twice produces no diff — guard every write so a no-op re-run leaves the file unchanged.
- **Minimal, surgical code.** Match existing style; one-line comments only where logic is non-obvious.

## Tests

Three layers, all must stay green (`bun run test`):

| Layer | File | Verifies |
| ----- | ---- | -------- |
| Unit | `test/unit.ts` | Pure helpers (jsonc, toml, paths, mcp-spawn). |
| Integration | `test/run.ts` | Sandboxed `$HOME`; runs `init` end-to-end and asserts the exact file shapes. Run directly with `TOKLESS_TEST=1 bun test/run.ts`. |
| Real-CLI | `test/real.ts` | Installs/queries the real agent CLI (`codex mcp list`, etc.). Skips cleanly if the CLI is absent. |

Add a test for every new config surface and run `bun run test` locally before opening a PR. (CI currently runs typecheck only.)

## PR

1. Fork, branch, implement.
2. `bun run test` — all green.
3. Open the PR with: what you added, the upstream docs link you followed, and `tokless doctor` (or `--dry-run`) output from your sandbox.
