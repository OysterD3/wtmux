import os from "node:os";
import path from "node:path";
import { describe, expect, it } from "vitest";
import { expandTilde, isAbsolutePath } from "../../src/paths.js";

describe("expandTilde", () => {
  it("expands a leading ~ to the home directory", () => {
    expect(expandTilde("~/foo")).toBe(path.join(os.homedir(), "foo"));
  });

  it("returns non-tilde paths unchanged", () => {
    expect(expandTilde("/abs/path")).toBe("/abs/path");
    expect(expandTilde("relative/path")).toBe("relative/path");
  });

  it("does not expand a mid-string tilde", () => {
    expect(expandTilde("/foo/~/bar")).toBe("/foo/~/bar");
  });
});

describe("isAbsolutePath", () => {
  it("returns true for POSIX absolute paths", () => {
    expect(isAbsolutePath("/a/b")).toBe(true);
  });

  it("returns false for relative and tilde paths", () => {
    expect(isAbsolutePath("a/b")).toBe(false);
    expect(isAbsolutePath("./a")).toBe(false);
    expect(isAbsolutePath("~/a")).toBe(false);
  });
});
