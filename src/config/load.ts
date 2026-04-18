import fs from "node:fs/promises";
import os from "node:os";
import { discoverConfigPath, type DiscoverySource } from "./discovery.js";
import { ConfigSchema, type Config } from "./schema.js";
import { WtmuxError } from "../errors.js";

export interface LoadInputs {
  cwd: string;
  home?: string;
  env?: NodeJS.ProcessEnv;
  explicit?: string;
}

export interface LoadedConfig {
  config: Config;
  path: string;
  source: DiscoverySource;
}

export async function loadConfig(inputs: LoadInputs): Promise<LoadedConfig | null> {
  const home = inputs.home ?? os.homedir();
  const env = inputs.env ?? process.env;
  const discovered = await discoverConfigPath({
    explicit: inputs.explicit,
    env,
    cwd: inputs.cwd,
    home,
  });
  if (!discovered) return null;

  const raw = await fs.readFile(discovered.path, "utf8");
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (err) {
    throw new WtmuxError(`invalid JSON in ${discovered.path}: ${(err as Error).message}`, "user");
  }

  const expanded = expandRepoPathsAgainstHome(parsed, home);
  const config = ConfigSchema.parse(expanded);
  return { config, path: discovered.path, source: discovered.source };
}

function expandRepoPathsAgainstHome(raw: unknown, home: string): unknown {
  if (!raw || typeof raw !== "object") return raw;
  const obj = raw as Record<string, unknown>;
  const groups = Array.isArray(obj.groups) ? obj.groups : undefined;
  if (!groups) return obj;
  return {
    ...obj,
    groups: groups.map((g) => {
      if (!g || typeof g !== "object") return g;
      const group = g as Record<string, unknown>;
      const repos = Array.isArray(group.repos)
        ? group.repos.map((r) => (typeof r === "string" ? expandWith(home, r) : r))
        : group.repos;
      return { ...group, repos };
    }),
  };
}

function expandWith(home: string, p: string): string {
  if (p === "~") return home;
  if (p.startsWith("~/")) return `${home}/${p.slice(2)}`;
  return p;
}
