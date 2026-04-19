import * as p from "@clack/prompts";
import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { DEFAULT_CONFIG } from "../config/defaults.js";
import { discoverConfigPath } from "../config/discovery.js";
import { saveConfig } from "../config/save.js";
import { ConfigSchema, type Config } from "../config/schema.js";
import { WtmuxError } from "../errors.js";
import { mainMenu } from "../tui/main-menu.js";

export interface ConfigFlowInput {
  cwd: string;
  env?: NodeJS.ProcessEnv;
  home?: string;
  explicit?: string;
}

export async function configFlow(input: ConfigFlowInput): Promise<void> {
  const home = input.home ?? os.homedir();
  const env = input.env ?? process.env;

  p.intro("wtmux config");

  const target = await resolveTarget({ explicit: input.explicit, env, cwd: input.cwd, home });
  if (target === null) {
    p.outro("No changes.");
    return;
  }

  const initial = await loadInitial(target);
  if (initial === null) {
    p.outro("No changes.");
    return;
  }

  const result = await mainMenu({ initial, targetPath: target, cwd: input.cwd });

  if (result.save) {
    try {
      await saveConfig(target, result.config);
      p.outro(`Saved to ${target}`);
    } catch (err) {
      p.log.error(`Failed to save: ${(err as Error).message}`);
      throw new WtmuxError(`config save failed: ${(err as Error).message}`, "internal");
    }
  } else {
    p.outro("Exited without saving.");
  }
}

async function resolveTarget(inputs: {
  explicit?: string;
  env: NodeJS.ProcessEnv;
  cwd: string;
  home: string;
}): Promise<string | null> {
  if (inputs.explicit) {
    return inputs.explicit;
  }

  const discovered = await discoverConfigPath({
    explicit: undefined,
    env: inputs.env,
    cwd: inputs.cwd,
    home: inputs.home,
  });
  if (discovered) return discovered.path;

  const proceed = await p.confirm({
    message: "No config found. Create one now?",
    initialValue: true,
  });
  if (p.isCancel(proceed) || !proceed) return null;

  const location = await p.select({
    message: "Where should the new config live?",
    options: [
      {
        value: "project",
        label: "Project",
        hint: `${path.join(inputs.cwd, ".wtmux.json")}`,
      },
      {
        value: "user",
        label: "User",
        hint: `${path.join(inputs.home, ".config", "wtmux", "config.json")}`,
      },
      { value: "cancel", label: "Cancel" },
    ],
  });
  if (p.isCancel(location) || location === "cancel") return null;

  if (location === "project") return path.join(inputs.cwd, ".wtmux.json");
  return path.join(inputs.home, ".config", "wtmux", "config.json");
}

async function loadInitial(target: string): Promise<Config | null> {
  let raw: string;
  try {
    raw = await fs.readFile(target, "utf8");
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") {
      return { ...DEFAULT_CONFIG, groups: [] };
    }
    p.log.error(`Could not read ${target}: ${(err as Error).message}`);
    return null;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (err) {
    p.log.error(`Invalid JSON in ${target}: ${(err as Error).message}`);
    const fallback = await p.confirm({
      message: "Start with an empty config instead? (existing file will only be overwritten on save)",
      initialValue: false,
    });
    if (p.isCancel(fallback) || !fallback) return null;
    return { ...DEFAULT_CONFIG, groups: [] };
  }

  const result = ConfigSchema.safeParse(parsed);
  if (!result.success) {
    p.log.error(
      `Schema errors in ${target}:\n${result.error.issues
        .map((i) => `  • ${i.path.join(".")}: ${i.message}`)
        .join("\n")}`,
    );
    const fallback = await p.confirm({
      message: "Start with an empty config instead? (existing file will only be overwritten on save)",
      initialValue: false,
    });
    if (p.isCancel(fallback) || !fallback) return null;
    return { ...DEFAULT_CONFIG, groups: [] };
  }

  return result.data;
}
