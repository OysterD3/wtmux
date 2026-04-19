import path from "node:path";
import { getToplevel } from "../git.js";
import { expandTilde } from "../paths.js";

export function validateGroupName(existingNames: readonly string[], name: string): string | undefined {
  const trimmed = name.trim();
  if (trimmed.length === 0) return "Name cannot be empty";
  if (trimmed.length > 64) return "Name is too long (max 64 characters)";
  if (existingNames.includes(trimmed)) return `A group named "${trimmed}" already exists`;
  return undefined;
}

export function validateWorktreePattern(pattern: string): string | undefined {
  const trimmed = pattern.trim();
  if (trimmed.length === 0) return undefined;
  if (!trimmed.includes("{name}")) return "Pattern must contain {name}";
  return undefined;
}

export interface RepoPathResult {
  ok: boolean;
  resolved?: string;
  error?: string;
}

export async function resolveRepoPath(raw: string): Promise<RepoPathResult> {
  const trimmed = raw.trim();
  if (trimmed.length === 0) return { ok: false, error: "Path cannot be empty" };

  const expanded = expandTilde(trimmed);
  if (!path.isAbsolute(expanded)) {
    return { ok: false, error: "Path must be absolute or start with ~/" };
  }

  const toplevel = await getToplevel(expanded);
  if (!toplevel) {
    return { ok: false, error: `Not a git repository: ${expanded}` };
  }

  return { ok: true, resolved: toplevel };
}
