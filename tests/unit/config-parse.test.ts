import { describe, expect, it } from "vitest";
import {
  formatCommaList,
  formatLaunchCommand,
  parseCommaList,
  parseLaunchCommand,
} from "../../src/tui/parse.js";

describe("parseCommaList", () => {
  it("splits on commas and trims", () => {
    expect(parseCommaList("node_modules, .env, .env.local")).toEqual([
      "node_modules",
      ".env",
      ".env.local",
    ]);
  });

  it("returns empty array for empty input", () => {
    expect(parseCommaList("")).toEqual([]);
    expect(parseCommaList("   ")).toEqual([]);
  });

  it("filters empty tokens from trailing/consecutive commas", () => {
    expect(parseCommaList("a,,b,")).toEqual(["a", "b"]);
  });
});

describe("formatCommaList", () => {
  it("joins with ', '", () => {
    expect(formatCommaList(["a", "b", "c"])).toBe("a, b, c");
  });

  it("returns empty string for empty list", () => {
    expect(formatCommaList([])).toBe("");
  });
});

describe("parseLaunchCommand", () => {
  it("splits on whitespace", () => {
    expect(parseLaunchCommand("code .")).toEqual(["code", "."]);
    expect(parseLaunchCommand("claude")).toEqual(["claude"]);
  });

  it("collapses multiple spaces", () => {
    expect(parseLaunchCommand("  claude   --resume  ")).toEqual(["claude", "--resume"]);
  });

  it("returns empty array for blank input", () => {
    expect(parseLaunchCommand("")).toEqual([]);
    expect(parseLaunchCommand("   ")).toEqual([]);
  });
});

describe("formatLaunchCommand", () => {
  it("joins with single space", () => {
    expect(formatLaunchCommand(["claude"])).toBe("claude");
    expect(formatLaunchCommand(["code", "."])).toBe("code .");
  });
});
