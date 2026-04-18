import fs from "node:fs/promises";
import { getToplevel } from "./git.js";
import type { Config, Group } from "./config/schema.js";

export type GroupResolution =
  | { kind: "group"; group: Group; primary: string }
  | { kind: "single"; repo: string }
  | { kind: "outside" };

export interface ResolveInputs {
  cwd: string;
  config: Config;
  groupFlag: string | undefined;
}

export async function resolveGroup(inputs: ResolveInputs): Promise<GroupResolution> {
  const { cwd, config, groupFlag } = inputs;

  if (groupFlag) {
    const g = config.groups.find((x) => x.name === groupFlag);
    if (!g) throw new Error(`--group "${groupFlag}" does not match any configured group`);
    const primary = await determinePrimary(cwd, g);
    return { kind: "group", group: g, primary };
  }

  const top = await getToplevel(cwd);
  if (!top) return { kind: "outside" };

  const realTop = await fs.realpath(top);
  const matches = await Promise.all(
    config.groups.map(async (g) => {
      const reals = await Promise.all(g.repos.map((r) => realpathSafe(r)));
      return reals.includes(realTop) ? g : null;
    }),
  );
  const found = matches.filter((g): g is Group => g !== null);

  if (found.length === 0) return { kind: "single", repo: realTop };
  if (found.length > 1) {
    throw new Error(
      `cwd matches more than one group (${found.map((g) => g.name).join(", ")}) — pass --group`,
    );
  }
  const group = found[0]!;
  const primary = await determinePrimary(cwd, group);
  return { kind: "group", group, primary };
}

export async function determinePrimary(cwd: string, group: Group): Promise<string> {
  const top = await getToplevel(cwd);
  if (top) {
    const real = await fs.realpath(top);
    const match = await matchInRepos(real, group.repos);
    if (match) return match;
  }
  return group.repos[0]!;
}

async function matchInRepos(candidate: string, repos: readonly string[]): Promise<string | null> {
  for (const r of repos) {
    const real = await realpathSafe(r);
    if (real === candidate) return r;
  }
  return null;
}

async function realpathSafe(p: string): Promise<string> {
  try {
    return await fs.realpath(p);
  } catch {
    return p;
  }
}
