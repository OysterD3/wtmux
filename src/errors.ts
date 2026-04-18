export type ErrorKind = "user" | "precondition" | "internal";

export class WtmuxError extends Error {
  readonly kind: ErrorKind;

  constructor(message: string, kind: ErrorKind) {
    super(message);
    this.kind = kind;
    this.name = "WtmuxError";
  }
}

export function exitCodeFor(err: unknown): number {
  if (err instanceof WtmuxError) {
    switch (err.kind) {
      case "user":
        return 1;
      case "precondition":
        return 2;
      case "internal":
        return 3;
    }
  }
  return 3;
}
