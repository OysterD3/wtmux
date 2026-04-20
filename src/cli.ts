import { defineCommand, runCommand } from "citty";
import { fileURLToPath } from "node:url";
import { realpathSync } from "node:fs";
import { createCommand } from "./commands/create.js";
import { rmCommand } from "./commands/rm.js";
import { lsCommand } from "./commands/ls.js";
import { configCommand } from "./commands/config.js";
import { exitCodeFor } from "./errors.js";
import pkg from "../package.json" with { type: "json" };

export const version: string = pkg.version;

// citty throws "Unknown command" for any non-flag arg not in subCommands.
// We detect whether the user is invoking a subcommand and route manually to
// avoid the conflict between positional <name> and subcommand routing.
const KNOWN_SUBCOMMANDS = new Set(["rm", "ls", "config"]);

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

  // Handle bare --help to show top-level usage including subcommands.
  if (rawArgs.length === 0 || (rawArgs.length === 1 && rawArgs[0] === "--help")) {
    process.stdout.write(
      [
        `wtmux v${version} — coordinated git worktrees across sibling repos`,
        ``,
        `Usage:`,
        `  wtmux <name> [-- agent-args...]   Create coordinated worktrees named <name>`,
        `  wtmux rm <name>                   Remove coordinated worktrees named <name>`,
        `  wtmux ls                          List worktrees across the group`,
        `  wtmux config                      Interactively edit wtmux config`,
        ``,
        `Subcommands:`,
        `  rm      Remove a coordinated worktree set`,
        `  ls      List worktrees across the group`,
        `  config  Interactively edit wtmux config`,
        ``,
        `Flags:`,
        `  -c, --config <path>    Override config discovery`,
        `  -g, --group <name>     Override auto-detected group`,
        `  -b, --base <branch>    Override the base branch (create only)`,
        `  -n, --dry-run          Print the plan without mutating`,
        `      --no-launch        Skip launching the agent at the end of create`,
        `  -f, --force            rm only: skip dirty/unpushed guards`,
        `  -v, --verbose          Extra logging`,
        `      --version          Print version`,
        `      --help             Print help`,
        ``,
      ].join("\n"),
    );
    process.exit(0);
  }

  // Handle `config --help` explicitly since citty 0.1.x passes --help into run() instead of intercepting it.
  if (firstArg === "config" && rawArgs.includes("--help")) {
    process.stdout.write(
      [
        `wtmux config — Interactively edit wtmux config`,
        ``,
        `Usage:`,
        `  wtmux config [options]`,
        ``,
        `Options:`,
        `  --config <path>   Path to config file`,
        `  -v, --verbose     Extra logging`,
        `  --help            Show this help`,
        ``,
      ].join("\n"),
    );
    process.exit(0);
  }

  if (firstArg !== undefined && KNOWN_SUBCOMMANDS.has(firstArg)) {
    // Route to a dedicated command that only has subCommands defined.
    const subCmd = defineCommand({
      meta: { name: "wtmux", version, description: "Coordinated git worktrees across sibling repos" },
      subCommands: { rm: rmCommand, ls: lsCommand, config: configCommand },
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
