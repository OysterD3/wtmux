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
