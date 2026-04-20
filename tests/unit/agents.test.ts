import { describe, expect, it } from "vitest";
import { AGENT_REGISTRY, BASENAME_ALIAS, type AgentId } from "../../src/agents.js";

describe("AGENT_REGISTRY", () => {
  it("contains exactly the six built-in agents", () => {
    const expected: AgentId[] = ["claude", "codex", "cursor", "code", "opencode", "qoder"];
    expect(Object.keys(AGENT_REGISTRY).sort()).toEqual([...expected].sort());
  });

  it("claude and codex both use --add-dir per sibling", () => {
    expect(AGENT_REGISTRY.claude).toEqual({ kind: "flag", args: ["--add-dir", "{path}"] });
    expect(AGENT_REGISTRY.codex).toEqual({ kind: "flag", args: ["--add-dir", "{path}"] });
  });

  it("cursor and code both use --add per sibling", () => {
    expect(AGENT_REGISTRY.cursor).toEqual({ kind: "flag", args: ["--add", "{path}"] });
    expect(AGENT_REGISTRY.code).toEqual({ kind: "flag", args: ["--add", "{path}"] });
  });

  it("opencode and qoder have no multi-root support", () => {
    expect(AGENT_REGISTRY.opencode).toEqual({ kind: "none" });
    expect(AGENT_REGISTRY.qoder).toEqual({ kind: "none" });
  });
});

describe("BASENAME_ALIAS", () => {
  it("maps qodercli to qoder", () => {
    expect(BASENAME_ALIAS.qodercli).toBe("qoder");
  });
});
