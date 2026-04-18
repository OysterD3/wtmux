import { describe, expect, it } from "vitest";
import { buildLaunchArgv } from "../../src/launch.js";

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
});
