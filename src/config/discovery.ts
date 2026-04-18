import fs from "node:fs/promises";
import path from "node:path";
import { WtmuxError } from "../errors.js";

export type DiscoverySource = "flag" | "env" | "walk" | "xdg";

export interface DiscoveryInputs {
  explicit?: string;
  env: NodeJS.ProcessEnv;
  cwd: string;
  home: string;
}

export interface DiscoveryResult {
  path: string;
  source: DiscoverySource;
}

async function exists(p: string): Promise<boolean> {
  try {
    await fs.access(p);
    return true;
  } catch {
    return false;
  }
}

export async function discoverConfigPath(inputs: DiscoveryInputs): Promise<DiscoveryResult | null> {
  const { explicit, env, cwd, home } = inputs;

  if (explicit) {
    if (!(await exists(explicit))) {
      throw new WtmuxError(`config file not found: ${explicit}`, "user");
    }
    return { path: explicit, source: "flag" };
  }

  const envPath = env.WTMUX_CONFIG;
  if (envPath) {
    if (!(await exists(envPath))) {
      throw new WtmuxError(`WTMUX_CONFIG not found: ${envPath}`, "user");
    }
    return { path: envPath, source: "env" };
  }

  let dir = path.resolve(cwd);
  while (true) {
    const candidate = path.join(dir, ".wtmux.json");
    if (await exists(candidate)) {
      return { path: candidate, source: "walk" };
    }
    const parent = path.dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }

  const xdg = path.join(home, ".config", "wtmux", "config.json");
  if (await exists(xdg)) {
    return { path: xdg, source: "xdg" };
  }

  return null;
}
