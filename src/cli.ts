#!/usr/bin/env node
import "./bootstrap.js"; // populates agent + tool registries before any command runs
import { runInit } from "./commands/init.js";
import { runUpdate } from "./commands/update.js";
import { runDoctor } from "./commands/doctor.js";
import { runDisable, runUninstall } from "./commands/disable.js";
import { setVerbose, log } from "./util/logger.js";
import { c } from "./util/colors.js";
import { agentIds, toolIds } from "./core/registry.js";
import { toklessVersion } from "./util/version.js";
import type { AgentId, ToolId } from "./core/ids.js";

interface ParsedArgs {
  cmd: string;
  flags: Record<string, string | boolean>;
  positional: string[];
}

function parseArgs(argv: string[]): ParsedArgs {
  const positional: string[] = [];
  const flags: Record<string, string | boolean> = {};
  let cmd = "";
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a.startsWith("--")) {
      const eq = a.indexOf("=");
      if (eq !== -1) flags[a.slice(2, eq)] = a.slice(eq + 1);
      else if (argv[i + 1] && !argv[i + 1].startsWith("-")) {
        flags[a.slice(2)] = argv[++i];
      } else {
        flags[a.slice(2)] = true;
      }
    } else if (a.startsWith("-") && a.length === 2) {
      flags[a.slice(1)] = true;
    } else if (!cmd) {
      cmd = a;
    } else {
      positional.push(a);
    }
  }
  return { cmd, flags, positional };
}

const HELP = `${c.bold(c.cyan("tokless"))} — token-saving for AI coding agents (Claude Code, OpenCode, Codex)

${c.bold("Usage:")}
  ${c.cyan("tokless")}              Install + wire everything (default; safe to re-run)
  ${c.cyan("tokless update")}       Show version diff and upgrade the 4 tools
  ${c.cyan("tokless doctor")}       Show what's wired up; warn about anything broken
  ${c.cyan("tokless uninstall")}    Remove everything tokless ever touched

${c.bold("Flags:")}
  --agents <list>     Limit to a subset: claude,opencode,codex
  --dry-run           Show what would change without writing anything
  --verbose           Show every step

${c.gray("Docs: https://github.com/HoangP8/tokless")}`;

async function main(): Promise<number> {
  const { cmd, flags } = parseArgs(process.argv.slice(2));
  if (flags.verbose === true) setVerbose(true);

  if (flags.version === true || flags.v === true || cmd === "version") {
    console.log(toklessVersion());
    return 0;
  }
  if (flags.help === true || cmd === "help") {
    console.log(HELP);
    return 0;
  }
  // Bare `tokless` (no command) defaults to `init` — install + wire everything.
  const command = cmd && cmd.length > 0 ? cmd : "init";

  let agents: AgentId[] | undefined;
  let tools: ToolId[] | undefined;
  try {
    agents = parseList(flags.agents, agentIds()) as AgentId[] | undefined;
    tools = parseList(flags.tools, toolIds()) as ToolId[] | undefined;
  } catch (err) {
    log.err((err as Error).message);
    return 2;
  }
  const opts = {
    agents,
    tools,
    yes: flags.yes === true,
    dryRun: flags["dry-run"] === true || flags["dryrun"] === true,
    verbose: flags.verbose === true,
  };

  try {
    if (command === "init") return await runInit(opts);
    if (command === "update") return await runUpdate(opts);
    if (command === "doctor") return await runDoctor({ offline: flags.offline === true });
    if (command === "disable") return await runDisable(opts);
    if (command === "uninstall") return await runUninstall(opts);
    if (command === "self-update") {
      const { runSelfUpdate } = await import("./commands/self-update.js");
      return await runSelfUpdate();
    }
    console.log(HELP);
    log.err(`Unknown command: ${command}`);
    return 1;
  } catch (err) {
    log.err((err as Error).message);
    if (flags.verbose === true) console.error((err as Error).stack);
    return 1;
  }
}

function parseList(raw: string | boolean | undefined, allowed: string[]): string[] | undefined {
  if (typeof raw !== "string") return undefined;
  const items = raw
    .split(",")
    .map((s) => s.trim().toLowerCase())
    .filter(Boolean);
  const invalid = items.filter((i) => !allowed.includes(i));
  if (invalid.length) {
    throw new Error(`Invalid value(s): ${invalid.join(", ")}. Allowed: ${allowed.join(", ")}`);
  }
  return items;
}

main().then((code) => process.exit(code));
