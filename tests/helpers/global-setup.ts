import { execa } from "execa";

export default async function setup(): Promise<void> {
  await execa("pnpm", ["build"], { stdio: "inherit" });
}
