import { describe, expect, it } from "vitest";
import { ConfigSchema } from "../../src/config/schema.js";

describe("ConfigSchema", () => {
  it("accepts a minimal valid config", () => {
    const result = ConfigSchema.safeParse({
      groups: [{ name: "g", repos: ["/abs/a", "/abs/b"] }],
    });
    expect(result.success).toBe(true);
  });

  it("applies defaults for symlinkDirectories, worktreePathPattern, launchCommand", () => {
    const parsed = ConfigSchema.parse({
      groups: [{ name: "g", repos: ["/abs/a", "/abs/b"] }],
    });
    expect(parsed.symlinkDirectories).toEqual(["node_modules", ".env"]);
    expect(parsed.worktreePathPattern).toBe(".worktrees/{name}");
    expect(parsed.launchCommand).toEqual(["claude"]);
  });

  it("rejects relative paths in group.repos", () => {
    const result = ConfigSchema.safeParse({
      groups: [{ name: "g", repos: ["rel/path", "/abs/b"] }],
    });
    expect(result.success).toBe(false);
  });

  it("rejects a group with fewer than 2 repos (coordination needs siblings)", () => {
    const result = ConfigSchema.safeParse({
      groups: [{ name: "g", repos: ["/abs/a"] }],
    });
    expect(result.success).toBe(false);
  });

  it("rejects duplicate group names", () => {
    const result = ConfigSchema.safeParse({
      groups: [
        { name: "g", repos: ["/abs/a", "/abs/b"] },
        { name: "g", repos: ["/abs/c", "/abs/d"] },
      ],
    });
    expect(result.success).toBe(false);
  });

  it("allows group-level symlinkDirectories override", () => {
    const parsed = ConfigSchema.parse({
      symlinkDirectories: ["node_modules"],
      groups: [
        { name: "g", repos: ["/abs/a", "/abs/b"], symlinkDirectories: ["node_modules", ".env.local"] },
      ],
    });
    expect(parsed.groups[0]!.symlinkDirectories).toEqual(["node_modules", ".env.local"]);
  });
});

describe("ConfigSchema — agent + addDirArgs", () => {
  it("accepts a valid top-level agent", () => {
    const parsed = ConfigSchema.parse({
      agent: "codex",
      groups: [{ name: "g", repos: ["/abs/a", "/abs/b"] }],
    });
    expect(parsed.agent).toBe("codex");
  });

  it("rejects an unknown agent value", () => {
    const result = ConfigSchema.safeParse({
      agent: "claude-code",
      groups: [{ name: "g", repos: ["/abs/a", "/abs/b"] }],
    });
    expect(result.success).toBe(false);
  });

  it("accepts a valid addDirArgs template containing {path}", () => {
    const parsed = ConfigSchema.parse({
      addDirArgs: ["--add-dir", "{path}"],
      groups: [{ name: "g", repos: ["/abs/a", "/abs/b"] }],
    });
    expect(parsed.addDirArgs).toEqual(["--add-dir", "{path}"]);
  });

  it("rejects addDirArgs without any {path} token", () => {
    const result = ConfigSchema.safeParse({
      addDirArgs: ["--foo", "bar"],
      groups: [{ name: "g", repos: ["/abs/a", "/abs/b"] }],
    });
    expect(result.success).toBe(false);
  });

  it("rejects an empty addDirArgs array", () => {
    const result = ConfigSchema.safeParse({
      addDirArgs: [],
      groups: [{ name: "g", repos: ["/abs/a", "/abs/b"] }],
    });
    expect(result.success).toBe(false);
  });

  it("allows per-group agent + addDirArgs override", () => {
    const parsed = ConfigSchema.parse({
      groups: [
        {
          name: "g",
          repos: ["/abs/a", "/abs/b"],
          agent: "cursor",
          addDirArgs: ["--dir={path}"],
        },
      ],
    });
    expect(parsed.groups[0]!.agent).toBe("cursor");
    expect(parsed.groups[0]!.addDirArgs).toEqual(["--dir={path}"]);
  });

  it("treats both fields as undefined when absent", () => {
    const parsed = ConfigSchema.parse({
      groups: [{ name: "g", repos: ["/abs/a", "/abs/b"] }],
    });
    expect(parsed.agent).toBeUndefined();
    expect(parsed.addDirArgs).toBeUndefined();
  });
});
