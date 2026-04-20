import { describe, expect, it } from "vitest";
import { buildLaunchArgv } from "../../src/launch.js";
import type { ResolvedStrategy } from "../../src/agents.js";
import { WtmuxError } from "../../src/errors.js";

describe("buildLaunchArgv", () => {
  it("injects flag-strategy args per sibling", () => {
    const strategy: ResolvedStrategy = {
      kind: "flag",
      args: ["--add-dir", "{path}"],
      source: "basename",
    };
    const argv = buildLaunchArgv({
      launchCommand: ["claude"],
      siblingWorktrees: ["/x/wt-a", "/x/wt-b"],
      strategy,
    });
    expect(argv).toEqual(["claude", "--add-dir", "/x/wt-a", "--add-dir", "/x/wt-b"]);
  });

  it("supports single-token templates (--dir={path})", () => {
    const strategy: ResolvedStrategy = {
      kind: "flag",
      args: ["--dir={path}"],
      source: "addDirArgs",
    };
    const argv = buildLaunchArgv({
      launchCommand: ["mytool"],
      siblingWorktrees: ["/x/wt-a"],
      strategy,
    });
    expect(argv).toEqual(["mytool", "--dir=/x/wt-a"]);
  });

  it("appends siblings positionally for positional-fallback strategy", () => {
    const strategy: ResolvedStrategy = { kind: "positional", source: "fallback" };
    const argv = buildLaunchArgv({
      launchCommand: ["code", "."],
      siblingWorktrees: ["/x/wt-a"],
      strategy,
    });
    expect(argv).toEqual(["code", ".", "/x/wt-a"]);
  });

  it("forwards extraArgs after the injected sibling args", () => {
    const strategy: ResolvedStrategy = {
      kind: "flag",
      args: ["--add-dir", "{path}"],
      source: "basename",
    };
    const argv = buildLaunchArgv({
      launchCommand: ["claude"],
      siblingWorktrees: ["/x/wt-a"],
      strategy,
      extraArgs: ["--verbose"],
    });
    expect(argv).toEqual(["claude", "--add-dir", "/x/wt-a", "--verbose"]);
  });

  it("returns launchCommand unchanged when siblings is empty (flag)", () => {
    const strategy: ResolvedStrategy = {
      kind: "flag",
      args: ["--add-dir", "{path}"],
      source: "basename",
    };
    const argv = buildLaunchArgv({
      launchCommand: ["claude"],
      siblingWorktrees: [],
      strategy,
    });
    expect(argv).toEqual(["claude"]);
  });

  it("returns launchCommand unchanged when siblings is empty (positional)", () => {
    const strategy: ResolvedStrategy = { kind: "positional", source: "fallback" };
    const argv = buildLaunchArgv({
      launchCommand: ["my-tool"],
      siblingWorktrees: [],
      strategy,
    });
    expect(argv).toEqual(["my-tool"]);
  });

  it("throws when called with a `none` strategy (caller must short-circuit)", () => {
    const strategy: ResolvedStrategy = {
      kind: "none",
      source: "basename",
      agentId: "opencode",
    };
    expect(() =>
      buildLaunchArgv({
        launchCommand: ["opencode"],
        siblingWorktrees: ["/x/wt"],
        strategy,
      }),
    ).toThrow(WtmuxError);
  });
});
