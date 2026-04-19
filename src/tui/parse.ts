export function parseCommaList(input: string): string[] {
  return input
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
}

export function formatCommaList(items: readonly string[]): string {
  return items.join(", ");
}

export function parseLaunchCommand(input: string): string[] {
  return input
    .trim()
    .split(/\s+/)
    .filter((s) => s.length > 0);
}

export function formatLaunchCommand(argv: readonly string[]): string {
  return argv.join(" ");
}
