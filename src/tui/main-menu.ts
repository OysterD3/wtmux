import * as p from "@clack/prompts";
import { createGroupWizard } from "./create-group.js";
import { deleteGroupWizard } from "./delete-group.js";
import { editGroupWizard } from "./edit-group.js";
import type { Config } from "../config/schema.js";

export interface MainMenuInput {
  initial: Config;
  targetPath: string;
  cwd: string;
}

export interface MainMenuResult {
  config: Config;
  save: boolean;
}

export async function mainMenu(input: MainMenuInput): Promise<MainMenuResult> {
  let current = input.initial;
  let dirty = false;

  while (true) {
    p.note(
      renderSummary(current, input.targetPath, dirty),
      "wtmux config",
    );

    const options = [
      { value: "create", label: "Create a new group" },
      ...(current.groups.length > 0
        ? [
            { value: "edit", label: "Edit a group" },
            { value: "delete", label: "Delete a group" },
          ]
        : []),
      { value: "save", label: dirty ? "Save & exit" : "Exit" },
      { value: "discard", label: dirty ? "Discard changes & exit" : "Cancel" },
    ];

    const action = await p.select({
      message: "What do you want to do?",
      options,
    });

    if (p.isCancel(action)) {
      if (!dirty) return { config: current, save: false };
      const confirm = await p.confirm({
        message: "Unsaved changes — save before exiting?",
        initialValue: true,
      });
      if (p.isCancel(confirm)) continue;
      return { config: current, save: confirm };
    }

    switch (action) {
      case "create": {
        const next = await createGroupWizard(current, input.cwd);
        if (next !== null) {
          current = next;
          dirty = true;
        }
        break;
      }
      case "edit": {
        const next = await editGroupWizard(current, input.cwd);
        if (next !== null) {
          current = next;
          dirty = true;
        }
        break;
      }
      case "delete": {
        const next = await deleteGroupWizard(current);
        if (next !== null) {
          current = next;
          dirty = true;
        }
        break;
      }
      case "save":
        return { config: current, save: dirty };
      case "discard":
        return { config: current, save: false };
    }
  }
}

function renderSummary(config: Config, targetPath: string, dirty: boolean): string {
  const lines = [`Editing: ${targetPath}${dirty ? "  (unsaved changes)" : ""}`, ""];
  if (config.groups.length === 0) {
    lines.push("Groups: (no groups yet)");
  } else {
    lines.push("Groups:");
    for (const g of config.groups) {
      lines.push(`  • ${g.name} (${g.repos.length} repos)`);
    }
  }
  return lines.join("\n");
}
