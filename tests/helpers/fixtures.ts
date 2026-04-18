import { execa } from "execa";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";

export async function makeTmpDir(label = "wtmux-test-"): Promise<string> {
  return fs.mkdtemp(path.join(os.tmpdir(), label));
}

export async function initRepo(dir: string, branch = "main"): Promise<void> {
  await fs.mkdir(dir, { recursive: true });
  await execa("git", ["init", "-b", branch], { cwd: dir });
  await execa("git", ["config", "user.email", "test@example.com"], { cwd: dir });
  await execa("git", ["config", "user.name", "Test"], { cwd: dir });
  await execa("git", ["config", "commit.gpgsign", "false"], { cwd: dir });
  await fs.writeFile(path.join(dir, "README.md"), "# test\n");
  await execa("git", ["add", "README.md"], { cwd: dir });
  await execa("git", ["commit", "-m", "initial"], { cwd: dir });
}

export async function writeFile(filePath: string, content: string): Promise<void> {
  await fs.mkdir(path.dirname(filePath), { recursive: true });
  await fs.writeFile(filePath, content);
}
