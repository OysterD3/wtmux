import { describe, expect, it } from "vitest";
import { generateRandomName, generateWithRetry } from "../../src/random-name.js";

describe("generateRandomName", () => {
  it("returns names matching wt/<adj>-<noun>", () => {
    for (let i = 0; i < 50; i++) {
      expect(generateRandomName()).toMatch(/^wt\/[a-z]+-[a-z]+$/);
    }
  });

  it("produces a reasonable variety of names", () => {
    const names = new Set<string>();
    for (let i = 0; i < 100; i++) names.add(generateRandomName());
    expect(names.size).toBeGreaterThan(50);
  });
});

describe("generateWithRetry", () => {
  it("returns the first generated name when nothing exists", async () => {
    const name = await generateWithRetry(async () => false);
    expect(name).toMatch(/^wt\/[a-z]+-[a-z]+$/);
  });

  it("retries when the generated name exists, then succeeds", async () => {
    let calls = 0;
    const exists = async (): Promise<boolean> => {
      calls++;
      return calls < 3;
    };
    const name = await generateWithRetry(exists);
    expect(calls).toBe(3);
    expect(name).toMatch(/^wt\/[a-z]+-[a-z]+$/);
  });

  it("throws after max attempts all collide", async () => {
    await expect(generateWithRetry(async () => true, 3)).rejects.toThrow(
      /could not generate/i,
    );
  });

  it("defaults max to 10", async () => {
    let calls = 0;
    await expect(
      generateWithRetry(async () => {
        calls++;
        return true;
      }),
    ).rejects.toThrow();
    expect(calls).toBe(10);
  });
});
