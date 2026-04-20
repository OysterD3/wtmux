import type { Dirent } from "node:fs";
import fs from "node:fs/promises";
import path from "node:path";

async function isGitRepo(dir: string): Promise<boolean> {
  try {
    const stat = await fs.stat(path.join(dir, ".git"));
    return stat.isDirectory() || stat.isFile();
  } catch {
    return false;
  }
}

async function collectGitRepos(dir: string, exclude?: string): Promise<string[]> {
  let entries: Dirent[];
  try {
    entries = await fs.readdir(dir, { withFileTypes: true });
  } catch {
    return [];
  }

  const results: string[] = [];
  for (const entry of entries) {
    if (!entry.isDirectory()) continue;
    const full = path.join(dir, entry.name);
    if (exclude && full === exclude) continue;
    if (await isGitRepo(full)) results.push(full);
  }
  return results;
}

export async function detectSiblings(cwd: string): Promise<string[]> {
  const resolved = path.resolve(cwd);

  if (await isGitRepo(resolved)) {
    const parent = path.dirname(resolved);
    if (parent === resolved) return [];
    return collectGitRepos(parent, resolved);
  }

  return collectGitRepos(resolved);
}
