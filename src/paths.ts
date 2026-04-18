import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";

export function expandTilde(p: string): string {
  if (p === "~") return os.homedir();
  if (p.startsWith("~/")) return path.join(os.homedir(), p.slice(2));
  return p;
}

export function isAbsolutePath(p: string): boolean {
  return path.isAbsolute(p);
}

export function expandWorktreePath(repo: string, pattern: string, name: string): string {
  const rendered = pattern.replaceAll("{name}", name);
  return path.isAbsolute(rendered) ? rendered : path.join(repo, rendered);
}

export async function pathExists(p: string): Promise<boolean> {
  try {
    await fs.lstat(p);
    return true;
  } catch {
    return false;
  }
}
