import * as p from "@clack/prompts";
import { AGENT_REGISTRY, type AgentId } from "../agents.js";
import { parseLaunchCommand } from "./parse.js";

const UNSET = "__unset__";
const CUSTOM = "__custom__";

const BUILTIN_OPTIONS = (Object.keys(AGENT_REGISTRY) as AgentId[]).map((id) => ({
  value: id,
  label: AGENT_REGISTRY[id].kind === "none" ? `${id} (no multi-root)` : id,
}));

const SELECT_OPTIONS = [
  { value: UNSET, label: "auto-detect (from launchCommand)" },
  ...BUILTIN_OPTIONS,
  { value: CUSTOM, label: "custom (configure addDirArgs)" },
];

export type AgentChoiceResult =
  | { cancelled: true }
  | {
      cancelled: false;
      agent: AgentId | undefined;
      addDirArgs: string[] | undefined;
    };

export interface PromptAgentChoiceOptions {
  message: string;
  currentAgent?: AgentId;
  currentAddDirArgs?: readonly string[];
}

export async function promptAgentChoice(
  options: PromptAgentChoiceOptions,
): Promise<AgentChoiceResult> {
  const { message, currentAgent, currentAddDirArgs } = options;
  const initialValue = currentAgent ?? (currentAddDirArgs ? CUSTOM : UNSET);

  const choice = await p.select({
    message,
    options: SELECT_OPTIONS,
    initialValue,
  });
  if (p.isCancel(choice)) return { cancelled: true };

  if (choice === UNSET) {
    return { cancelled: false, agent: undefined, addDirArgs: undefined };
  }
  if (choice === CUSTOM) {
    const input = await p.text({
      message: "addDirArgs template (space-separated, use {path} for sibling)",
      placeholder: "--add-dir {path}",
      initialValue: currentAddDirArgs
        ? currentAddDirArgs.join(" ")
        : "--add-dir {path}",
      validate: (v) => (v.includes("{path}") ? undefined : "must contain {path}"),
    });
    if (p.isCancel(input)) return { cancelled: true };
    return {
      cancelled: false,
      agent: undefined,
      addDirArgs: parseLaunchCommand(input as string),
    };
  }
  return {
    cancelled: false,
    agent: choice as AgentId,
    addDirArgs: undefined,
  };
}
