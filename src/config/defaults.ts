import type { Config } from "./schema.js";

export const DEFAULT_CONFIG: Config = {
  symlinkDirectories: ["node_modules", ".env"],
  worktreePathPattern: ".worktrees/{name}",
  launchCommand: ["claude"],
  groups: [],
};
