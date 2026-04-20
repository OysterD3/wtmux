import { spawnSync } from "node:child_process";
import path from "node:path";

export function isClaudeCommand(argv0: string): boolean {
  return path.basename(argv0) === "claude";
}

export interface BuildLaunchArgvInput {
  launchCommand: readonly string[];
  siblingWorktrees: readonly string[];
  extraArgs?: readonly string[];
}

export function buildLaunchArgv(input: BuildLaunchArgvInput): string[] {
  const { launchCommand, siblingWorktrees, extraArgs = [] } = input;
  const base = [...launchCommand];
  const siblingArgs: string[] = [];
  const cmd = launchCommand[0];
  if (cmd !== undefined && isClaudeCommand(cmd)) {
    for (const p of siblingWorktrees) siblingArgs.push("--add-dir", p);
  } else {
    for (const p of siblingWorktrees) siblingArgs.push(p);
  }
  return [...base, ...siblingArgs, ...extraArgs];
}

export interface ExecLaunchInput {
  argv: string[];
  cwd: string;
}

export function execLaunch(input: ExecLaunchInput): never {
  const [cmd, ...rest] = input.argv;
  if (!cmd) {
    process.stderr.write("[wtmux] launchCommand is empty — nothing to exec\n");
    process.exit(2);
  }
  const result = spawnSync(cmd, rest, { cwd: input.cwd, stdio: "inherit" });
  process.exit(result.status ?? 1);
}
