# tokless

One command to install a full set of token-saving tools into your AI coding agents — Claude Code, OpenCode, and Codex — cutting token use without hurting how the agent performs.

| Tool | What it does |
| ---- | ------------ |
| [RTK](https://github.com/rtk-ai/rtk) | Compresses noisy bash/tool output before it reaches the model |
| [Caveman](https://github.com/JuliusBrussee/caveman) | Trims the agent's prose to terse, token-efficient output |
| [CodeGraph](https://github.com/colbymchenry/codegraph) | MCP server — the agent queries a code graph instead of reading whole files |
| [Context-Mode](https://github.com/mksglu/context-mode) | MCP server — runs data-heavy work in a sandbox, returns only the relevant slice |

Each tool is installed from its **official source** and wired into each agent exactly as that tool's own docs prescribe. tokless only ever *adds* its entries — your existing config is merged, never overwritten — and re-running is idempotent.

## Install

A single self-contained binary. No Node, no npm, no Python to install first — the runtime is embedded.

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/HoangP8/tokless/main/scripts/install.sh | bash
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/HoangP8/tokless/main/scripts/install.ps1 | iex
```

The binary installs to `~/.local/bin` (macOS/Linux). If that's already on your `PATH` — the default on most systems — `tokless` works immediately. Otherwise the installer adds one line to your shell rc so new terminals pick it up.

If a tool needs a runtime you don't have (e.g. `npm` for the MCP servers), tokless asks before installing it — nothing is installed behind your back.

## Commands

```
tokless              Install + wire everything (default; safe to re-run)
tokless update       Show the version diff and upgrade the four tools
tokless doctor       Show what's wired up; warn about anything broken
tokless uninstall    Remove everything tokless ever touched
tokless self-update  Update the tokless CLI itself
```

Flags:

```
--agents <list>   Limit to a subset: claude,opencode,codex
--dry-run         Show what would change without writing anything
--verbose         Show every step
```

```bash
tokless                              # interactive: pick agents, wire all four tools
tokless --agents opencode --dry-run  # preview, no writes
tokless doctor
```

After running it, restart your agents so they pick up the new MCP servers and config.

## License

MIT © [HoangP8](https://github.com/HoangP8)
