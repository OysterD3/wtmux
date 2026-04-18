import { defineCommand } from "citty";
import { loadConfig } from "../config/load.js";
import { rmFlow } from "../flows/rm.js";
import { setVerbose } from "../log.js";
import type { Config } from "../config/schema.js";

export const rmCommand = defineCommand({
  meta: { name: "rm", description: "Remove coordinated worktrees" },
  args: {
    name: { type: "positional", required: true },
    config: { type: "string" },
    group: { type: "string" },
    "dry-run": { type: "boolean", description: "Plan without mutating" },
    force: { type: "boolean", description: "Skip dirty/stash/unpushed guards" },
    verbose: { type: "boolean", alias: "v" },
  },
  async run({ args }) {
    setVerbose(Boolean(args.verbose));
    const loaded = await loadConfig({
      cwd: process.cwd(),
      env: process.env,
      explicit: args.config,
    });
    const config: Config = loaded?.config ?? {
      symlinkDirectories: ["node_modules", ".env"],
      worktreePathPattern: ".worktrees/{name}",
      launchCommand: ["claude"],
      groups: [],
    };

    const result = await rmFlow({
      name: args.name,
      cwd: process.cwd(),
      config,
      groupFlag: args.group,
      dryRun: Boolean(args["dry-run"]),
      force: Boolean(args.force),
    });

    for (const r of result.removed) process.stdout.write(`removed ${r.wtPath}\n`);
    for (const s of result.skipped) process.stderr.write(`skipped ${s.wtPath} (${s.reason})\n`);
  },
});
