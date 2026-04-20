import { z } from "zod";

const AbsolutePath = z.string().refine((p) => p.startsWith("/"), {
  message: "must be an absolute path (tilde-expanded by loader before validation)",
});

const AgentIdSchema = z.enum(["claude", "codex", "cursor", "code", "opencode", "qoder"]);

const AddDirArgsSchema = z
  .array(z.string().min(1))
  .min(1)
  .refine((a) => a.some((t) => t.includes("{path}")), {
    message: "addDirArgs must contain at least one entry with {path}",
  });

const GroupSchema = z.object({
  name: z.string().min(1),
  repos: z.array(AbsolutePath).min(2),
  symlinkDirectories: z.array(z.string().min(1)).optional(),
  worktreePathPattern: z.string().min(1).optional(),
  launchCommand: z.array(z.string().min(1)).min(1).optional(),
  agent: AgentIdSchema.optional(),
  addDirArgs: AddDirArgsSchema.optional(),
});

export const ConfigSchema = z
  .object({
    $schema: z.string().optional(),
    symlinkDirectories: z.array(z.string().min(1)).default(["node_modules", ".env"]),
    worktreePathPattern: z.string().min(1).default(".worktrees/{name}"),
    launchCommand: z.array(z.string().min(1)).min(1).default(["claude"]),
    agent: AgentIdSchema.optional(),
    addDirArgs: AddDirArgsSchema.optional(),
    groups: z.array(GroupSchema).default([]),
  })
  .superRefine((cfg, ctx) => {
    const seen = new Set<string>();
    for (const [i, g] of cfg.groups.entries()) {
      if (seen.has(g.name)) {
        ctx.addIssue({
          code: "custom",
          path: ["groups", i, "name"],
          message: `duplicate group name: ${g.name}`,
        });
      }
      seen.add(g.name);
    }
  });

export type Config = z.infer<typeof ConfigSchema>;
export type Group = z.infer<typeof GroupSchema>;
