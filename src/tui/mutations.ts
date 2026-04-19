import { ConfigSchema, type Config, type Group } from "../config/schema.js";

export function addGroup(config: Config, group: Group): Config {
  if (config.groups.some((g) => g.name === group.name)) {
    throw new Error(`duplicate group name: ${group.name}`);
  }
  return validated({ ...config, groups: [...config.groups, group] });
}

export function removeGroup(config: Config, name: string): Config {
  if (!config.groups.some((g) => g.name === name)) {
    throw new Error(`group not found: ${name}`);
  }
  return validated({ ...config, groups: config.groups.filter((g) => g.name !== name) });
}

export function renameGroup(config: Config, oldName: string, newName: string): Config {
  if (!config.groups.some((g) => g.name === oldName)) {
    throw new Error(`group not found: ${oldName}`);
  }
  if (oldName !== newName && config.groups.some((g) => g.name === newName)) {
    throw new Error(`duplicate group name: ${newName}`);
  }
  return validated({
    ...config,
    groups: config.groups.map((g) => (g.name === oldName ? { ...g, name: newName } : g)),
  });
}

export function updateGroup(config: Config, name: string, patch: Partial<Group>): Config {
  if (!config.groups.some((g) => g.name === name)) {
    throw new Error(`group not found: ${name}`);
  }
  return validated({
    ...config,
    groups: config.groups.map((g) => (g.name === name ? { ...g, ...patch } : g)),
  });
}

export function addReposToGroup(config: Config, groupName: string, repos: readonly string[]): Config {
  const target = config.groups.find((g) => g.name === groupName);
  if (!target) throw new Error(`group not found: ${groupName}`);
  const existing = new Set(target.repos);
  const merged = [...target.repos];
  for (const r of repos) {
    if (!existing.has(r)) {
      merged.push(r);
      existing.add(r);
    }
  }
  return updateGroup(config, groupName, { repos: merged });
}

export function removeReposFromGroup(
  config: Config,
  groupName: string,
  repos: readonly string[],
): Config {
  const target = config.groups.find((g) => g.name === groupName);
  if (!target) throw new Error(`group not found: ${groupName}`);
  const toRemove = new Set(repos);
  const remaining = target.repos.filter((r) => !toRemove.has(r));
  if (remaining.length < 2) {
    throw new Error(`group "${groupName}" must have at least 2 repos after removal`);
  }
  return updateGroup(config, groupName, { repos: remaining });
}

function validated(config: Config): Config {
  return ConfigSchema.parse(config);
}
