let verbose = false;

export function setVerbose(v: boolean): void {
  verbose = v;
}

export function info(message: string): void {
  process.stderr.write(`[wtmux] ${message}\n`);
}

export function warn(message: string): void {
  process.stderr.write(`[wtmux] ${message}\n`);
}

export function error(message: string): void {
  process.stderr.write(`[wtmux] ${message}\n`);
}

export function debug(message: string): void {
  if (verbose) process.stderr.write(`[wtmux:debug] ${message}\n`);
}
