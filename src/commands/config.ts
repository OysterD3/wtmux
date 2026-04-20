import { defineCommand } from "citty";
import { configFlow } from "../flows/config.js";
import { setVerbose } from "../log.js";

export const configCommand = defineCommand({
  meta: { name: "config", description: "Interactively edit wtmux config" },
  args: {
    config: { type: "string", alias: "c", description: "Path to config file" },
    verbose: { type: "boolean", alias: "v" },
  },
  async run({ args }) {
    setVerbose(Boolean(args.verbose));
    await configFlow({
      cwd: process.cwd(),
      env: process.env,
      explicit: args.config,
    });
  },
});
