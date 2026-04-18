import { describe, expect, it } from "vitest";
import { WtmuxError, exitCodeFor } from "../../src/errors.js";

describe("WtmuxError", () => {
  it("tags errors with a code used by the top-level handler", () => {
    const err = new WtmuxError("bad config", "user");
    expect(err.kind).toBe("user");
    expect(exitCodeFor(err)).toBe(1);
  });

  it("maps precondition to exit code 2", () => {
    expect(exitCodeFor(new WtmuxError("no group", "precondition"))).toBe(2);
  });

  it("maps internal to exit code 3", () => {
    expect(exitCodeFor(new WtmuxError("oops", "internal"))).toBe(3);
  });

  it("maps unknown errors to 3", () => {
    expect(exitCodeFor(new Error("something"))).toBe(3);
  });
});
