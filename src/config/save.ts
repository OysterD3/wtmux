import fs from "node:fs/promises";
import path from "node:path";
import { ConfigSchema, type Config } from "./schema.js";

export async function saveConfig(target: string, config: Config): Promise<void> {
  const validated = ConfigSchema.parse(config);
  const json = JSON.stringify(validated, null, 2) + "\n";

  await fs.mkdir(path.dirname(target), { recursive: true });

  const tmpPath = target + ".tmp";
  await fs.writeFile(tmpPath, json, "utf8");
  await fs.rename(tmpPath, target);
}
