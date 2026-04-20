import { describe, expect, it } from "vitest";
import { AGENT_REGISTRY, BASENAME_ALIAS, resolveStrategy, type AgentId, type ResolvedStrategy } from "../../src/agents.js";

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

describe("resolveStrategy", () => {
  it("returns addDirArgs strategy when addDirArgs is set", () => {
    const got = resolveStrategy({
      launchCommand: ["claude"],
      addDirArgs: ["--foo", "{path}"],
    });
    expect(got).toEqual({
      kind: "flag",
      args: ["--foo", "{path}"],
      source: "addDirArgs",
    } satisfies ResolvedStrategy);
  });

  it("addDirArgs beats agent and emits a warning", () => {
    const warnings: string[] = [];
    const got = resolveStrategy({
      launchCommand: ["claude"],
      agent: "claude",
      addDirArgs: ["--foo", "{path}"],
      warn: (m) => warnings.push(m),
    });
    expect(got.source).toBe("addDirArgs");
    expect(warnings).toEqual([`addDirArgs overrides agent "claude"`]);
  });

  it("uses agent registry when agent is set and addDirArgs is absent", () => {
    const got = resolveStrategy({ launchCommand: ["whatever"], agent: "codex" });
    expect(got).toEqual({
      kind: "flag",
      args: ["--add-dir", "{path}"],
      source: "agent",
    } satisfies ResolvedStrategy);
  });

  it("returns `none` strategy for opencode agent with agentId echoed back", () => {
    const got = resolveStrategy({ launchCommand: ["whatever"], agent: "opencode" });
    expect(got).toEqual({ kind: "none", source: "agent", agentId: "opencode" });
  });

  it("falls back to basename detection for bare claude", () => {
    const got = resolveStrategy({ launchCommand: ["claude"] });
    expect(got).toEqual({
      kind: "flag",
      args: ["--add-dir", "{path}"],
      source: "basename",
    });
  });

  it("basename detection strips absolute paths", () => {
    const got = resolveStrategy({ launchCommand: ["/usr/local/bin/codex"] });
    expect(got).toEqual({
      kind: "flag",
      args: ["--add-dir", "{path}"],
      source: "basename",
    });
  });

  it("basename detection strips home-relative paths", () => {
    const got = resolveStrategy({ launchCommand: ["~/.local/bin/cursor"] });
    expect(got).toEqual({
      kind: "flag",
      args: ["--add", "{path}"],
      source: "basename",
    });
  });

  it("qodercli basename aliases to qoder (kind: none)", () => {
    const got = resolveStrategy({ launchCommand: ["qodercli"] });
    expect(got).toEqual({ kind: "none", source: "basename", agentId: "qoder" });
  });

  it("unknown basename falls through to positional", () => {
    const got = resolveStrategy({ launchCommand: ["my-custom-tool"] });
    expect(got).toEqual({ kind: "positional", source: "fallback" } satisfies ResolvedStrategy);
  });

  it("claude-code is not claude — falls through to positional", () => {
    const got = resolveStrategy({ launchCommand: ["claude-code"] });
    expect(got.source).toBe("fallback");
    expect(got.kind).toBe("positional");
  });
});
