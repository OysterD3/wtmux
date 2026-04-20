import { defineCommand } from "citty";
import { loadConfig } from "../config/load.js";
import { DEFAULT_CONFIG } from "../config/defaults.js";
import { createFlow } from "../flows/create.js";
import { setVerbose } from "../log.js";
import type { Config } from "../config/schema.js";

export const createCommand = defineCommand({
  meta: { name: "wtmux", description: "Create coordinated worktrees and launch claude" },
  args: {
    name: { type: "positional", required: false, description: "Worktree/branch name" },
    config: { type: "string", alias: "c", description: "Path to config file" },
    group: { type: "string", alias: "g", description: "Group name override" },
    "dry-run": { type: "boolean", alias: "n", description: "Plan without mutating" },
    base: { type: "string", alias: "b", description: "Override base branch" },
    // NOTE: citty's parser treats --no-X as negating flag X, so we declare
    // "launch" with default:true and the user passes --no-launch to set it false.
    launch: { type: "boolean", default: true, description: "Launch after creating (use --no-launch to skip)" },
    verbose: { type: "boolean", alias: "v" },
  },
  async run({ args, rawArgs }) {
    setVerbose(Boolean(args.verbose));
    if (!args.name) {
      process.stderr.write("usage: wtmux <name> [-- claude-args...]\n");
      process.exit(1);
    }
    const loaded = await loadConfig({
      cwd: process.cwd(),
      env: process.env,
      explicit: args.config,
    });
    const config: Config = loaded?.config ?? DEFAULT_CONFIG;

    const extraArgs = extractPassthrough(rawArgs);
    await createFlow({
      name: args.name,
      cwd: process.cwd(),
      config,
      groupFlag: args.group,
      dryRun: Boolean(args["dry-run"]),
      noLaunch: !args.launch,
      extraArgs,
      baseOverride: args.base,
    });
  },
});

function extractPassthrough(rawArgs: string[]): string[] {
  const idx = rawArgs.indexOf("--");
  return idx >= 0 ? rawArgs.slice(idx + 1) : [];
}
