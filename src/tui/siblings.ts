import type { Dirent } from "node:fs";
import fs from "node:fs/promises";
import path from "node:path";

export async function detectSiblings(cwd: string): Promise<string[]> {
  const resolved = path.resolve(cwd);
  const parent = path.dirname(resolved);
  if (parent === resolved) return [];

  let entries: Dirent[];
  try {
    entries = await fs.readdir(parent, { withFileTypes: true });
  } catch {
    return [];
  }

  const results: string[] = [];
  for (const entry of entries) {
    if (!entry.isDirectory()) continue;
    const full = path.join(parent, entry.name);
    if (full === resolved) continue;
    const gitPath = path.join(full, ".git");
    try {
      const stat = await fs.stat(gitPath);
      if (stat.isDirectory()) results.push(full);
    } catch {
      // not a git repo
    }
  }
  return results;
}
