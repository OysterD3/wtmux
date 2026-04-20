import { describe, expect, it } from "vitest";
import { version } from "../src/cli.js";

describe("smoke", () => {
  it("exposes a version string", () => {
    expect(version).toBe("0.2.0");
  });
});
