import * as p from "@clack/prompts";
import { removeGroup } from "./mutations.js";
import type { Config } from "../config/schema.js";

export async function deleteGroupWizard(config: Config): Promise<Config | null> {
  if (config.groups.length === 0) {
    p.log.info("No groups to delete.");
    return null;
  }

  const pick = await p.select({
    message: "Which group to delete?",
    options: config.groups.map((g) => ({
      value: g.name,
      label: g.name,
      hint: `${g.repos.length} repos`,
    })),
  });
  if (p.isCancel(pick)) return null;

  const confirm = await p.confirm({
    message: `Delete "${pick}"? (Can still be undone via Discard & exit)`,
    initialValue: false,
  });
  if (p.isCancel(confirm) || !confirm) return null;

  const next = removeGroup(config, pick as string);
  p.log.success(`Deleted group "${pick}"`);
  return next;
}
