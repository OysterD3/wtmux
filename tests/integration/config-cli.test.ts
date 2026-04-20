import { execa } from "execa";
import path from "node:path";
import { describe, expect, it } from "vitest";

const BIN = path.resolve("dist/wtmux.js");

describe("cli config", () => {
  it("prints help for `wtmux config --help`", async () => {
    const result = await execa("node", [BIN, "config", "--help"]);
    expect(result.stdout).toMatch(/config/);
    expect(result.stdout).toMatch(/Interactively edit/i);
  });

  it("is listed in top-level help", async () => {
    const result = await execa("node", [BIN, "--help"]);
    expect(result.stdout).toMatch(/config/);
  });
});
