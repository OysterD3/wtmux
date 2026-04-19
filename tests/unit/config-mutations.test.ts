import { describe, expect, it } from "vitest";
import {
  addGroup,
  addReposToGroup,
  removeGroup,
  removeReposFromGroup,
  renameGroup,
  updateGroup,
} from "../../src/tui/mutations.js";
import type { Config, Group } from "../../src/config/schema.js";

function baseConfig(): Config {
  return {
    symlinkDirectories: ["node_modules", ".env"],
    worktreePathPattern: ".worktrees/{name}",
    launchCommand: ["claude"],
    groups: [],
  };
}

function group(name: string, repos: string[] = ["/abs/a", "/abs/b"]): Group {
  return { name, repos };
}

describe("addGroup", () => {
  it("appends a group", () => {
    const next = addGroup(baseConfig(), group("myapp"));
    expect(next.groups.map((g) => g.name)).toEqual(["myapp"]);
  });

  it("throws on duplicate name", () => {
    const cfg = addGroup(baseConfig(), group("myapp"));
    expect(() => addGroup(cfg, group("myapp"))).toThrow(/duplicate/i);
  });
});

describe("removeGroup", () => {
  it("removes a group by name", () => {
    const cfg = addGroup(addGroup(baseConfig(), group("a")), group("b"));
    const next = removeGroup(cfg, "a");
    expect(next.groups.map((g) => g.name)).toEqual(["b"]);
  });

  it("throws when group not found", () => {
    expect(() => removeGroup(baseConfig(), "nope")).toThrow(/not found/i);
  });
});

describe("renameGroup", () => {
  it("renames an existing group", () => {
    const cfg = addGroup(baseConfig(), group("old"));
    const next = renameGroup(cfg, "old", "new");
    expect(next.groups.map((g) => g.name)).toEqual(["new"]);
  });

  it("throws when renaming to a name that already exists", () => {
    const cfg = addGroup(addGroup(baseConfig(), group("a")), group("b"));
    expect(() => renameGroup(cfg, "a", "b")).toThrow(/duplicate/i);
  });

  it("throws when source does not exist", () => {
    expect(() => renameGroup(baseConfig(), "nope", "new")).toThrow(/not found/i);
  });
});

describe("updateGroup", () => {
  it("merges a patch into the named group", () => {
    const cfg = addGroup(baseConfig(), group("g"));
    const next = updateGroup(cfg, "g", { symlinkDirectories: ["node_modules"] });
    expect(next.groups[0]!.symlinkDirectories).toEqual(["node_modules"]);
  });
});

describe("addReposToGroup", () => {
  it("dedupes and appends", () => {
    const cfg = addGroup(baseConfig(), group("g", ["/abs/a", "/abs/b"]));
    const next = addReposToGroup(cfg, "g", ["/abs/b", "/abs/c"]);
    expect(next.groups[0]!.repos).toEqual(["/abs/a", "/abs/b", "/abs/c"]);
  });
});

describe("removeReposFromGroup", () => {
  it("removes selected repos", () => {
    const cfg = addGroup(baseConfig(), group("g", ["/abs/a", "/abs/b", "/abs/c"]));
    const next = removeReposFromGroup(cfg, "g", ["/abs/b"]);
    expect(next.groups[0]!.repos).toEqual(["/abs/a", "/abs/c"]);
  });

  it("throws when resulting repo count < 2 (schema min-2)", () => {
    const cfg = addGroup(baseConfig(), group("g", ["/abs/a", "/abs/b"]));
    expect(() => removeReposFromGroup(cfg, "g", ["/abs/b"])).toThrow(/at least 2/i);
  });
});
