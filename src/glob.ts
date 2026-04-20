import { glob } from "tinyglobby";

const GLOB_CHARS = /[*?[\]{}]/;

export async function expandSymlinkItems(
  repo: string,
  items: readonly string[],
): Promise<string[]> {
  const seen = new Set<string>();
  const expanded: string[] = [];

  for (const item of items) {
    if (!GLOB_CHARS.test(item)) {
      if (!seen.has(item)) {
        seen.add(item);
        expanded.push(item);
      }
      continue;
    }

    const matches = await glob(item, {
      cwd: repo,
      dot: true,
      onlyFiles: false,
      absolute: false,
    });

    for (const m of matches) {
      const normalized = m.replaceAll("\\", "/");
      if (!seen.has(normalized)) {
        seen.add(normalized);
        expanded.push(normalized);
      }
    }
  }

  return expanded;
}
