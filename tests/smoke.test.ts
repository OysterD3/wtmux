import { describe, expect, it } from "vitest";
import { version } from "../src/cli.js";
import pkg from "../package.json" with { type: "json" };

describe("smoke", () => {
  it("exposes a semver-shaped version string", () => {
    expect(version).toMatch(/^\d+\.\d+\.\d+(-[\w.]+)?$/);
  });

  it("matches package.json", () => {
    expect(version).toBe(pkg.version);
  });
});
