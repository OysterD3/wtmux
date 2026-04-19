import { describe, expect, it } from "vitest";
import {
  validateGroupName,
  validateWorktreePattern,
} from "../../src/tui/prompts.js";

describe("validateGroupName", () => {
  it("accepts a unique non-empty name", () => {
    expect(validateGroupName([], "myapp")).toBeUndefined();
  });

  it("rejects empty", () => {
    expect(validateGroupName([], "")).toMatch(/empty/i);
    expect(validateGroupName([], "   ")).toMatch(/empty/i);
  });

  it("rejects names over 64 chars", () => {
    expect(validateGroupName([], "x".repeat(65))).toMatch(/too long/i);
  });

  it("rejects duplicates", () => {
    expect(validateGroupName(["myapp"], "myapp")).toMatch(/already/i);
  });

  it("allows re-using the current name when editing (existing list excludes self)", () => {
    expect(validateGroupName([], "current")).toBeUndefined();
  });
});

describe("validateWorktreePattern", () => {
  it("accepts a pattern containing {name}", () => {
    expect(validateWorktreePattern(".worktrees/{name}")).toBeUndefined();
  });

  it("accepts empty (meaning: unset override)", () => {
    expect(validateWorktreePattern("")).toBeUndefined();
  });

  it("rejects a non-empty pattern missing {name}", () => {
    expect(validateWorktreePattern(".worktrees/fixed")).toMatch(/\{name\}/);
  });
});
