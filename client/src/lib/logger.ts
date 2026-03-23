type Level = "debug" | "info" | "warn" | "error";

const LEVELS: Record<Level, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

const MIN_LEVEL: Level = import.meta.env.PROD ? "info" : "debug";

function log(level: Level, namespace: string, message: string, context?: unknown) {
  if (LEVELS[level] < LEVELS[MIN_LEVEL]) return;

  const entry = {
    ts: new Date().toISOString(),
    level,
    ns: namespace,
    msg: message,
    ...(context !== undefined && { ctx: context }),
  };

  const fn = level === "error" ? console.error : level === "warn" ? console.warn : console.log;
  fn(`[${entry.ts}] ${level.toUpperCase()} [${namespace}]`, message, ...(context !== undefined ? [context] : []));
}

export function createLogger(namespace: string) {
  return {
    debug: (msg: string, ctx?: unknown) => log("debug", namespace, msg, ctx),
    info:  (msg: string, ctx?: unknown) => log("info",  namespace, msg, ctx),
    warn:  (msg: string, ctx?: unknown) => log("warn",  namespace, msg, ctx),
    error: (msg: string, ctx?: unknown) => log("error", namespace, msg, ctx),
  };
}
