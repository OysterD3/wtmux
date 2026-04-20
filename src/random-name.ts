const ADJECTIVES = [
  "brave", "calm", "eager", "fast", "gentle", "happy", "kind", "lively",
  "merry", "nimble", "odd", "plucky", "quick", "quiet", "regal", "rusty",
  "silver", "swift", "tame", "vivid", "witty", "warm", "bold", "cozy",
  "dapper", "fair", "fuzzy", "jolly", "lucky", "mild", "neat", "proud",
  "shiny", "sleek", "snug", "solid", "spry", "sunny", "tidy", "trim",
  "bright", "breezy", "crisp", "giddy", "grand", "jaunty", "keen", "lush",
  "mellow", "peppy", "zesty",
] as const;

const NOUNS = [
  "badger", "cactus", "comet", "crane", "dolphin", "ember", "falcon", "fern",
  "glacier", "harbor", "iris", "kelp", "lantern", "lynx", "maple", "meadow",
  "mist", "moss", "nebula", "otter", "pebble", "petal", "puffin", "quartz",
  "raven", "reef", "river", "saffron", "sage", "sparrow", "spruce", "star",
  "storm", "sunset", "tide", "tiger", "thistle", "tulip", "valley", "vine",
  "walnut", "wave", "willow", "wolf", "zephyr", "acorn", "birch", "cedar",
  "cinder", "dune", "eagle",
] as const;

export function generateRandomName(): string {
  const a = ADJECTIVES[Math.floor(Math.random() * ADJECTIVES.length)]!;
  const n = NOUNS[Math.floor(Math.random() * NOUNS.length)]!;
  return `wt/${a}-${n}`;
}

export async function generateWithRetry(
  exists: (name: string) => Promise<boolean>,
  max = 10,
): Promise<string> {
  for (let i = 0; i < max; i++) {
    const candidate = generateRandomName();
    if (!(await exists(candidate))) return candidate;
  }
  throw new Error(`could not generate a unique random name after ${max} attempts`);
}
