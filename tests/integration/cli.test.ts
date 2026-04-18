import { execa } from "execa";
import fs from "node:fs/promises";
import path from "node:path";
import { afterEach, beforeAll, describe, expect, it } from "vitest";
import { initRepo, makeTmpDir } from "../helpers/fixtures.js";

const BIN = path.resolve("dist/wtmux.js");

async function _ensureBuilt(): Promise<void> {
  try {
    await fs.access(BIN);
  } catch {
    await execa("pnpm", ["build"], { stdio: "inherit" });
  }
}

const tmpdirs: string[] = [];
const tmp = async () => {
  const d = await makeTmpDir();
  tmpdirs.push(d);
  return d;
};

beforeAll(async () => {
  // Always rebuild to pick up latest CLI code
  await execa("pnpm", ["build"], { stdio: "inherit" });
});

afterEach(async () => {
  while (tmpdirs.length) await fs.rm(tmpdirs.pop()!, { recursive: true, force: true });
});

describe("cli", () => {
  it("prints --version", async () => {
    const result = await execa("node", [BIN, "--version"]);
    expect(result.stdout).toMatch(/0\.1\.0/);
  });

  it("creates coordinated worktrees via `wtmux <name>`", async () => {
    const a = await tmp();
    const b = await tmp();
    await initRepo(a);
    await initRepo(b);
    const ra = await fs.realpath(a);
    const rb = await fs.realpath(b);
    await fs.writeFile(
      path.join(ra, ".wtmux.json"),
      JSON.stringify({ groups: [{ name: "g", repos: [ra, rb] }] }, null, 2),
    );

    const result = await execa("node", [BIN, "feat/cli", "--no-launch"], {
      cwd: ra,
      reject: false,
    });
    expect(result.exitCode).toBe(0);
    expect((await fs.stat(path.join(ra, ".worktrees/feat/cli"))).isDirectory()).toBe(true);
    expect((await fs.stat(path.join(rb, ".worktrees/feat/cli"))).isDirectory()).toBe(true);
  });

  it("removes via `wtmux rm <name>`", async () => {
    const a = await tmp();
    const b = await tmp();
    await initRepo(a);
    await initRepo(b);
    const ra = await fs.realpath(a);
    const rb = await fs.realpath(b);
    await fs.writeFile(
      path.join(ra, ".wtmux.json"),
      JSON.stringify({ groups: [{ name: "g", repos: [ra, rb] }] }, null, 2),
    );
    await execa("node", [BIN, "feat/gone", "--no-launch"], { cwd: ra });
    const result = await execa("node", [BIN, "rm", "feat/gone"], { cwd: ra, reject: false });
    expect(result.exitCode).toBe(0);
    await expect(fs.stat(path.join(ra, ".worktrees/feat/gone"))).rejects.toThrow();
  });

  it("exits non-zero with a clear message on precondition failure (no repo)", async () => {
    const bare = await tmp();
    const result = await execa("node", [BIN, "feat/x", "--no-launch"], {
      cwd: bare,
      reject: false,
    });
    expect(result.exitCode).not.toBe(0);
    expect(result.stderr).toMatch(/not inside any git repository/i);
  });
});
