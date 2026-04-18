import { defineCommand } from "citty";
import { loadConfig } from "../config/load.js";
import { DEFAULT_CONFIG } from "../config/defaults.js";
import { lsFlow, type LsRow } from "../flows/ls.js";
import { setVerbose } from "../log.js";
import type { Config } from "../config/schema.js";

export const lsCommand = defineCommand({
  meta: { name: "ls", description: "List coordinated worktrees in the current group" },
  args: {
    config: { type: "string" },
    group: { type: "string" },
    verbose: { type: "boolean", alias: "v" },
  },
  async run({ args }) {
    setVerbose(Boolean(args.verbose));
    const loaded = await loadConfig({
      cwd: process.cwd(),
      env: process.env,
      explicit: args.config,
    });
    const config: Config = loaded?.config ?? DEFAULT_CONFIG;

    const rows = await lsFlow({ cwd: process.cwd(), config, groupFlag: args.group });
    if (rows.length === 0) {
      process.stdout.write("(no coordinated worktrees)\n");
      return;
    }
    for (const row of rows) process.stdout.write(format(row) + "\n");
  },
});

function format(row: LsRow): string {
  const parts = row.perRepo.map((s) => {
    if (!s.present) return `${short(s.repo)}: —`;
    return `${short(s.repo)}: ${s.state}`;
  });
  return `${row.name.padEnd(24)} ${parts.join("   ")}`;
}

function short(repo: string): string {
  const parts = repo.split("/");
  return parts[parts.length - 1] ?? repo;
}
