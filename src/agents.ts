import path from "node:path";

export type AgentId = "claude" | "codex" | "cursor" | "code" | "opencode" | "qoder";

export type AgentStrategy =
  | { kind: "flag"; args: readonly string[] }
  | { kind: "none" };

export const AGENT_REGISTRY: Record<AgentId, AgentStrategy> = {
  claude: { kind: "flag", args: ["--add-dir", "{path}"] },
  codex: { kind: "flag", args: ["--add-dir", "{path}"] },
  cursor: { kind: "flag", args: ["--add", "{path}"] },
  code: { kind: "flag", args: ["--add", "{path}"] },
  opencode: { kind: "none" },
  qoder: { kind: "none" },
};

export const BASENAME_ALIAS: Record<string, AgentId> = {
  qodercli: "qoder",
};

export type ResolvedStrategy =
  | { kind: "flag"; args: readonly string[]; source: "addDirArgs" | "agent" | "basename" }
  | { kind: "none"; source: "agent" | "basename"; agentId: AgentId }
  | { kind: "positional"; source: "fallback" };

export interface ResolveStrategyInput {
  launchCommand: readonly string[];
  agent?: AgentId;
  addDirArgs?: readonly string[];
  warn?: (message: string) => void;
}

export function resolveStrategy(input: ResolveStrategyInput): ResolvedStrategy {
  const { launchCommand, agent, addDirArgs, warn } = input;

  if (addDirArgs && addDirArgs.length > 0) {
    if (agent) warn?.(`addDirArgs overrides agent "${agent}"`);
    return { kind: "flag", args: addDirArgs, source: "addDirArgs" };
  }

  if (agent) {
    const strategy = AGENT_REGISTRY[agent];
    return strategy.kind === "flag"
      ? { kind: "flag", args: strategy.args, source: "agent" }
      : { kind: "none", source: "agent", agentId: agent };
  }

  const argv0 = launchCommand[0];
  if (argv0) {
    const basename = path.basename(argv0);
    const resolved = (BASENAME_ALIAS[basename] ?? basename) as AgentId;
    const strategy = AGENT_REGISTRY[resolved];
    if (strategy) {
      return strategy.kind === "flag"
        ? { kind: "flag", args: strategy.args, source: "basename" }
        : { kind: "none", source: "basename", agentId: resolved };
    }
  }

  return { kind: "positional", source: "fallback" };
}

export function expandAddDirArgs(
  template: readonly string[],
  siblings: readonly string[],
): string[] {
  const out: string[] = [];
  for (const sibling of siblings) {
    for (const token of template) {
      out.push(token.replaceAll("{path}", sibling));
    }
  }
  return out;
}
