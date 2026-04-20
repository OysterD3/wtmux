import { spawnSync } from "node:child_process";
import { expandAddDirArgs, type ResolvedStrategy } from "./agents.js";
import { WtmuxError } from "./errors.js";

export interface BuildLaunchArgvInput {
  launchCommand: readonly string[];
  siblingWorktrees: readonly string[];
  strategy: ResolvedStrategy;
  extraArgs?: readonly string[];
}

export function buildLaunchArgv(input: BuildLaunchArgvInput): string[] {
  const { launchCommand, siblingWorktrees, strategy, extraArgs = [] } = input;

  if (strategy.kind === "none") {
    throw new WtmuxError(
      `buildLaunchArgv called with "none" strategy (agent: ${strategy.agentId}); callers must short-circuit the launch`,
      "internal",
    );
  }

  const siblingArgs =
    strategy.kind === "flag"
      ? expandAddDirArgs(strategy.args, siblingWorktrees)
      : [...siblingWorktrees];

  return [...launchCommand, ...siblingArgs, ...extraArgs];
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
