import * as p from "@clack/prompts";
import path from "node:path";
import { addGroup, updateGroup } from "./mutations.js";
import { detectSiblings } from "./siblings.js";
import { formatCommaList, formatLaunchCommand, parseCommaList, parseLaunchCommand } from "./parse.js";
import {
  resolveRepoPath,
  validateGroupName,
  validateWorktreePattern,
} from "./prompts.js";
import type { Config, Group } from "../config/schema.js";
import { promptAgentChoice } from "./agent-select.js";

export async function createGroupWizard(config: Config, cwd: string): Promise<Config | null> {
  const existingNames = config.groups.map((g) => g.name);

  const name = await p.text({
    message: "Group name",
    validate: (v) => validateGroupName(existingNames, v),
  });
  if (p.isCancel(name)) return null;

  const repos = await collectRepos(cwd);
  if (repos === null) return null;
  if (repos.length < 2) {
    p.log.error("A group needs at least 2 repos. Aborting.");
    return null;
  }

  let next = addGroup(config, { name: name.trim(), repos });

  const wantOverrides = await p.confirm({
    message: "Configure overrides for this group?",
    initialValue: false,
  });
  if (p.isCancel(wantOverrides)) return null;

  if (wantOverrides) {
    next = await collectOverrides(next, name.trim());
  }

  p.log.success(`Created group "${name.trim()}" (${repos.length} repos)`);
  return next;
}

async function collectRepos(cwd: string): Promise<string[] | null> {
  const siblings = await detectSiblings(cwd);

  let selected: string[] = [];
  if (siblings.length > 0) {
    const picks = await p.multiselect({
      message: "Which sibling repos?",
      options: siblings.map((s) => ({ value: s, label: path.basename(s), hint: s })),
      required: false,
    });
    if (p.isCancel(picks)) return null;
    selected = picks as string[];
  } else {
    p.log.info("No sibling git repos detected in the parent directory.");
  }

  const wantManual = await p.confirm({
    message: "Add any other repo paths manually?",
    initialValue: selected.length < 2,
  });
  if (p.isCancel(wantManual)) return null;

  if (wantManual) {
    while (true) {
      const raw = await p.text({
        message: `Repo path ${selected.length} entered; blank to finish`,
        placeholder: "~/code/another-repo",
      });
      if (p.isCancel(raw)) return null;
      if (raw.trim().length === 0) break;

      const result = await resolveRepoPath(raw);
      if (!result.ok) {
        p.log.error(result.error ?? "Invalid path");
        continue;
      }
      if (selected.includes(result.resolved!)) {
        p.log.warn("Already added.");
        continue;
      }
      selected.push(result.resolved!);
    }
  }

  return selected;
}

async function collectOverrides(config: Config, groupName: string): Promise<Config> {
  const group = config.groups.find((g) => g.name === groupName)!;

  const symlinks = await p.text({
    message: "Override symlinkDirectories? (comma-separated, blank to skip)",
    placeholder: formatCommaList(config.symlinkDirectories),
    initialValue: group.symlinkDirectories ? formatCommaList(group.symlinkDirectories) : "",
  });
  if (p.isCancel(symlinks)) return config;

  const pattern = await p.text({
    message: "Override worktreePathPattern? (blank to skip, must include {name})",
    placeholder: config.worktreePathPattern,
    initialValue: group.worktreePathPattern ?? "",
    validate: validateWorktreePattern,
  });
  if (p.isCancel(pattern)) return config;

  const launch = await p.text({
    message: "Override launchCommand? (space-separated, blank to skip)",
    placeholder: formatLaunchCommand(config.launchCommand),
    initialValue: group.launchCommand ? formatLaunchCommand(group.launchCommand) : "",
  });
  if (p.isCancel(launch)) return config;

  const agentResult = await promptAgentChoice({
    message: "Agent (sibling-injection strategy, optional)",
  });
  if (agentResult.cancelled) return config;

  const patch: Partial<Group> = {};
  if (symlinks.trim().length > 0) patch.symlinkDirectories = parseCommaList(symlinks);
  if (pattern.trim().length > 0) patch.worktreePathPattern = pattern.trim();
  if (launch.trim().length > 0) patch.launchCommand = parseLaunchCommand(launch);
  if (agentResult.agent !== undefined) patch.agent = agentResult.agent;
  if (agentResult.addDirArgs !== undefined) patch.addDirArgs = agentResult.addDirArgs;

  return Object.keys(patch).length > 0 ? updateGroup(config, groupName, patch) : config;
}
