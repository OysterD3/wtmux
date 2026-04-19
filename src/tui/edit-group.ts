import * as p from "@clack/prompts";
import path from "node:path";
import {
  addReposToGroup,
  removeReposFromGroup,
  renameGroup,
  updateGroup,
} from "./mutations.js";
import { detectSiblings } from "./siblings.js";
import {
  formatCommaList,
  formatLaunchCommand,
  parseCommaList,
  parseLaunchCommand,
} from "./parse.js";
import {
  resolveRepoPath,
  validateGroupName,
  validateWorktreePattern,
} from "./prompts.js";
import type { Config } from "../config/schema.js";

export async function editGroupWizard(config: Config, cwd: string): Promise<Config | null> {
  if (config.groups.length === 0) {
    p.log.info("No groups to edit.");
    return null;
  }

  const pick = await p.select({
    message: "Which group?",
    options: config.groups.map((g) => ({
      value: g.name,
      label: g.name,
      hint: `${g.repos.length} repos`,
    })),
  });
  if (p.isCancel(pick)) return null;
  const groupName = pick as string;

  let current = config;
  let activeName = groupName;

  while (true) {
    const field = await p.select({
      message: `Editing "${activeName}" — which field?`,
      options: [
        { value: "name", label: "Rename group" },
        { value: "repos", label: "Edit repos" },
        { value: "symlinks", label: "Override symlinkDirectories" },
        { value: "pattern", label: "Override worktreePathPattern" },
        { value: "launch", label: "Override launchCommand" },
        { value: "back", label: "← Back to main menu" },
      ],
    });
    if (p.isCancel(field) || field === "back") return current === config ? null : current;

    switch (field) {
      case "name": {
        const group = current.groups.find((g) => g.name === activeName)!;
        const others = current.groups.map((g) => g.name).filter((n) => n !== group.name);
        const newName = await p.text({
          message: "New name",
          initialValue: group.name,
          validate: (v) => validateGroupName(others, v),
        });
        if (p.isCancel(newName)) break;
        current = renameGroup(current, activeName, (newName as string).trim());
        activeName = (newName as string).trim();
        break;
      }
      case "repos": {
        const next = await editRepos(current, activeName, cwd);
        if (next !== null) current = next;
        break;
      }
      case "symlinks": {
        const group = current.groups.find((g) => g.name === activeName)!;
        const input = await p.text({
          message: "symlinkDirectories (comma-separated, blank to unset override)",
          placeholder: formatCommaList(current.symlinkDirectories),
          initialValue: group.symlinkDirectories ? formatCommaList(group.symlinkDirectories) : "",
        });
        if (p.isCancel(input)) break;
        const patch =
          (input as string).trim().length > 0
            ? { symlinkDirectories: parseCommaList(input as string) }
            : { symlinkDirectories: undefined };
        current = updateGroup(current, activeName, patch);
        break;
      }
      case "pattern": {
        const group = current.groups.find((g) => g.name === activeName)!;
        const input = await p.text({
          message: "worktreePathPattern (blank to unset override)",
          placeholder: current.worktreePathPattern,
          initialValue: group.worktreePathPattern ?? "",
          validate: validateWorktreePattern,
        });
        if (p.isCancel(input)) break;
        const patch =
          (input as string).trim().length > 0
            ? { worktreePathPattern: (input as string).trim() }
            : { worktreePathPattern: undefined };
        current = updateGroup(current, activeName, patch);
        break;
      }
      case "launch": {
        const group = current.groups.find((g) => g.name === activeName)!;
        const input = await p.text({
          message: "launchCommand (space-separated, blank to unset override)",
          placeholder: formatLaunchCommand(current.launchCommand),
          initialValue: group.launchCommand ? formatLaunchCommand(group.launchCommand) : "",
        });
        if (p.isCancel(input)) break;
        const patch =
          (input as string).trim().length > 0
            ? { launchCommand: parseLaunchCommand(input as string) }
            : { launchCommand: undefined };
        current = updateGroup(current, activeName, patch);
        break;
      }
    }
  }
}

async function editRepos(config: Config, groupName: string, cwd: string): Promise<Config | null> {
  let current = config;
  while (true) {
    const group = current.groups.find((g) => g.name === groupName)!;
    const action = await p.select({
      message: `Repos (${group.repos.length}): ${group.repos.map((r) => path.basename(r)).join(", ")}`,
      options: [
        { value: "add", label: "Add a repo" },
        {
          value: "remove",
          label: "Remove repos",
          hint: group.repos.length <= 2 ? "need at least 2" : "",
        },
        { value: "done", label: "← Done" },
      ],
    });
    if (p.isCancel(action) || action === "done") return current;

    if (action === "add") {
      const toAdd = await collectReposToAdd(cwd, group.repos);
      if (toAdd === null) return current;
      if (toAdd.length > 0) current = addReposToGroup(current, groupName, toAdd);
    } else if (action === "remove") {
      if (group.repos.length <= 2) {
        p.log.error("Cannot remove — a group must have at least 2 repos.");
        continue;
      }
      const toRemove = await p.multiselect({
        message: "Select repos to remove",
        options: group.repos.map((r) => ({ value: r, label: path.basename(r), hint: r })),
        required: false,
      });
      if (p.isCancel(toRemove)) return current;
      if ((toRemove as string[]).length > 0) {
        try {
          current = removeReposFromGroup(current, groupName, toRemove as string[]);
        } catch (err) {
          p.log.error((err as Error).message);
        }
      }
    }
  }
}

async function collectReposToAdd(
  cwd: string,
  alreadyIn: readonly string[],
): Promise<string[] | null> {
  const siblings = (await detectSiblings(cwd)).filter((s) => !alreadyIn.includes(s));

  const added: string[] = [];
  if (siblings.length > 0) {
    const picks = await p.multiselect({
      message: "Which sibling repos to add?",
      options: siblings.map((s) => ({ value: s, label: path.basename(s), hint: s })),
      required: false,
    });
    if (p.isCancel(picks)) return null;
    added.push(...(picks as string[]));
  }

  const wantManual = await p.confirm({
    message: "Add manual paths?",
    initialValue: added.length === 0,
  });
  if (p.isCancel(wantManual)) return null;

  if (wantManual) {
    while (true) {
      const raw = await p.text({
        message: "Repo path (blank to finish)",
        placeholder: "~/code/another-repo",
      });
      if (p.isCancel(raw)) return null;
      if ((raw as string).trim().length === 0) break;

      const result = await resolveRepoPath(raw as string);
      if (!result.ok) {
        p.log.error(result.error ?? "Invalid path");
        continue;
      }
      if (alreadyIn.includes(result.resolved!) || added.includes(result.resolved!)) {
        p.log.warn("Already added.");
        continue;
      }
      added.push(result.resolved!);
    }
  }

  return added;
}
