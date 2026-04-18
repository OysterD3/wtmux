import { defineConfig } from "tsup";

export default defineConfig({
  entry: { wtmux: "src/cli.ts" },
  format: ["esm"],
  target: "node20",
  outDir: "dist",
  outExtension: () => ({ js: ".js" }),
  clean: true,
  splitting: false,
  minify: false,
  sourcemap: false,
  banner: { js: "#!/usr/bin/env node" },
});
