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
