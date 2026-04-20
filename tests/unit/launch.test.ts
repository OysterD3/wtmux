import { describe, expect, it } from "vitest";
import { buildLaunchArgv, isClaudeCommand } from "../../src/launch.js";

describe("buildLaunchArgv", () => {
  it("appends --add-dir per sibling when launchCommand is claude", () => {
    const argv = buildLaunchArgv({
      launchCommand: ["claude"],
      siblingWorktrees: ["/x/wt-a", "/x/wt-b"],
    });
    expect(argv).toEqual(["claude", "--add-dir", "/x/wt-a", "--add-dir", "/x/wt-b"]);
  });

  it("appends siblings positionally when launchCommand is not claude", () => {
    const argv = buildLaunchArgv({
      launchCommand: ["code", "."],
      siblingWorktrees: ["/x/wt-a"],
    });
    expect(argv).toEqual(["code", ".", "/x/wt-a"]);
  });

  it("forwards extra args between wrapper and launch", () => {
    const argv = buildLaunchArgv({
      launchCommand: ["claude"],
      siblingWorktrees: ["/x/wt-a"],
      extraArgs: ["--verbose"],
    });
    expect(argv).toEqual(["claude", "--add-dir", "/x/wt-a", "--verbose"]);
  });

  it("does not append anything when there are no siblings (single-repo fallback)", () => {
    const argv = buildLaunchArgv({
      launchCommand: ["claude"],
      siblingWorktrees: [],
    });
    expect(argv).toEqual(["claude"]);
  });

  it("injects --add-dir when launchCommand is an absolute path to claude", () => {
    const argv = buildLaunchArgv({
      launchCommand: ["/usr/local/bin/claude"],
      siblingWorktrees: ["/x/wt-a"],
    });
    expect(argv).toEqual(["/usr/local/bin/claude", "--add-dir", "/x/wt-a"]);
  });

  it("injects --add-dir when launchCommand uses a home-relative claude path", () => {
    const argv = buildLaunchArgv({
      launchCommand: ["~/.local/bin/claude"],
      siblingWorktrees: ["/x/wt-a"],
    });
    expect(argv).toEqual(["~/.local/bin/claude", "--add-dir", "/x/wt-a"]);
  });

  it("does not inject --add-dir for claude-code (different tool)", () => {
    const argv = buildLaunchArgv({
      launchCommand: ["claude-code"],
      siblingWorktrees: ["/x/wt-a"],
    });
    expect(argv).toEqual(["claude-code", "/x/wt-a"]);
  });

  it("does not inject --add-dir for fakeclaude", () => {
    const argv = buildLaunchArgv({
      launchCommand: ["fakeclaude"],
      siblingWorktrees: ["/x/wt-a"],
    });
    expect(argv).toEqual(["fakeclaude", "/x/wt-a"]);
  });
});

describe("isClaudeCommand", () => {
  it("matches bare claude", () => {
    expect(isClaudeCommand("claude")).toBe(true);
  });

  it("matches an absolute path whose basename is claude", () => {
    expect(isClaudeCommand("/usr/local/bin/claude")).toBe(true);
  });

  it("rejects claude-code", () => {
    expect(isClaudeCommand("claude-code")).toBe(false);
  });

  it("rejects an absolute path ending in claude-dev", () => {
    expect(isClaudeCommand("/opt/bin/claude-dev")).toBe(false);
  });

  it("rejects a non-claude command", () => {
    expect(isClaudeCommand("code")).toBe(false);
  });
});
