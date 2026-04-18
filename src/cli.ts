import { defineCommand, runCommand } from "citty";
import { fileURLToPath } from "node:url";
import { realpathSync } from "node:fs";
import { createCommand } from "./commands/create.js";
import { rmCommand } from "./commands/rm.js";
import { lsCommand } from "./commands/ls.js";
import { exitCodeFor } from "./errors.js";

export const version = "0.1.0";

// citty throws "Unknown command" for any non-flag arg not in subCommands.
// We detect whether the user is invoking a subcommand and route manually to
// avoid the conflict between positional <name> and subcommand routing.
const KNOWN_SUBCOMMANDS = new Set(["rm", "ls"]);

function firstNonFlag(args: string[]): string | undefined {
  return args.find((a) => !a.startsWith("-"));
}

const rawArgs = process.argv.slice(2);
const firstArg = firstNonFlag(rawArgs);

async function dispatch(): Promise<void> {
  // Handle --version ourselves to guarantee stdout output (consola may vary by TTY).
  if (rawArgs.length === 1 && rawArgs[0] === "--version") {
    process.stdout.write(`${version}\n`);
    process.exit(0);
  }

  if (firstArg !== undefined && KNOWN_SUBCOMMANDS.has(firstArg)) {
    // Route to a dedicated command that only has subCommands defined.
    const subCmd = defineCommand({
      meta: { name: "wtmux", version, description: "Coordinated git worktrees across sibling repos" },
      subCommands: { rm: rmCommand, ls: lsCommand },
    });
    await runCommand(subCmd, { rawArgs });
  } else {
    // Route to the create command that has positional args but no subCommands.
    const createCmd = defineCommand({
      meta: { name: "wtmux", version, description: "Coordinated git worktrees across sibling repos" },
      args: createCommand.args,
      async run(ctx) {
        await createCommand.run!(ctx);
      },
    });
    await runCommand(createCmd, { rawArgs });
  }
}

// Only run dispatch when this module is the process entry point, not when
// imported by tests. Resolve symlinks so global installs (pnpm add -g, npm link)
// work correctly.
/* c8 ignore next */
const selfPath = fileURLToPath(import.meta.url);
let entryPath = process.argv[1] ?? "";
try {
  entryPath = realpathSync(entryPath);
} catch {
  // leave as-is; comparison below will fail cleanly
}
/* c8 ignore next */
if (entryPath === selfPath) {
  dispatch().catch((err) => {
    const message = err instanceof Error ? err.message : String(err);
    process.stderr.write(`[wtmux] ${message}\n`);
    process.exit(exitCodeFor(err));
  });
}
