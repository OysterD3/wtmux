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
