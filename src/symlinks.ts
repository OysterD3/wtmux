import fs from "node:fs/promises";
import path from "node:path";

export type SymlinkAction =
  | "linked"
  | "skipped-no-source"
  | "skipped-target-exists";

export interface SymlinkResult {
  item: string;
  action: SymlinkAction;
}

export interface ReplicateInputs {
  repo: string;
  wt: string;
  items: readonly string[];
}

export async function replicateSymlinks(inputs: ReplicateInputs): Promise<SymlinkResult[]> {
  const results: SymlinkResult[] = [];
  for (const item of inputs.items) {
    const source = path.join(inputs.repo, item);
    const target = path.join(inputs.wt, item);

    let sourceStat;
    try {
      sourceStat = await fs.stat(source);
    } catch (err) {
      const code = (err as NodeJS.ErrnoException).code;
      if (code === "ENOENT") {
        results.push({ item, action: "skipped-no-source" });
        continue;
      }
      throw err;
    }

    try {
      await fs.lstat(target);
      results.push({ item, action: "skipped-target-exists" });
      continue;
    } catch (err) {
      const code = (err as NodeJS.ErrnoException).code;
      if (code !== "ENOENT") throw err;
      // target missing — good
    }

    await fs.mkdir(path.dirname(target), { recursive: true });
    await fs.symlink(source, target, sourceStat.isDirectory() ? "dir" : "file");
    results.push({ item, action: "linked" });
  }
  return results;
}

export async function removeSymlinks(wt: string, items: readonly string[]): Promise<void> {
  for (const item of items) {
    const target = path.join(wt, item);
    try {
      const stat = await fs.lstat(target);
      if (stat.isSymbolicLink()) await fs.unlink(target);
    } catch (err) {
      const code = (err as NodeJS.ErrnoException).code;
      if (code !== "ENOENT") throw err;
      // nothing to remove
    }
  }
}
