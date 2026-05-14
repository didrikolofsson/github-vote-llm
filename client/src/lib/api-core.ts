import { z } from "zod";
import { createLogger } from "./logger";

const BASE = "/v1";
const logger = createLogger("api");

let accessToken: string | null = null;
let onRefresh: (() => Promise<string | null>) | null = null;

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

export function getAccessToken(): string | null {
  return accessToken;
}

export function setOnRefresh(fn: (() => Promise<string | null>) | null): void {
  onRefresh = fn;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public body?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export function formatApiError(err: unknown): string {
  if (err instanceof ApiError) {
    if (
      typeof err.body === "object" &&
      err.body !== null &&
      "error" in err.body
    ) {
      return String((err.body as { error: string }).error);
    }
    return err.message;
  }
  if (err instanceof z.ZodError) {
    const parts = err.issues.map(
      (i) => `${i.path.filter(Boolean).join(".") || "response"}: ${i.message}`,
    );
    return `Validation failed: ${parts.join("; ")}`;
  }
  return err instanceof Error ? err.message : String(err);
}

export async function request<S extends z.ZodSchema>(
  path: string,
  options: RequestInit & { schema: S; skipAuth?: boolean },
): Promise<z.infer<S>> {
  const { schema, skipAuth, ...init } = options;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(init.headers as Record<string, string>),
  };

  if (!skipAuth && accessToken) {
    headers["Authorization"] = `Bearer ${accessToken}`;
  }

  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers,
  });

  const text = await res.text();
  let data: unknown;
  try {
    data = text ? JSON.parse(text) : undefined;
  } catch {
    data = text;
  }

  if (!res.ok) {
    const errBody =
      typeof data === "object" && data !== null && "error" in data
        ? (data as { error: string }).error
        : data;
    throw new ApiError(
      typeof errBody === "string" ? errBody : `Request failed: ${res.status}`,
      res.status,
      data,
    );
  }

  return schema.parseAsync(data).catch((err) => {
    logger.error("Response validation failed", { path, err });
    throw err;
  });
}

export async function requestWithRefresh<S extends z.ZodSchema>(
  path: string,
  options: RequestInit & { schema: S; skipAuth?: boolean },
): Promise<z.infer<S>> {
  try {
    return await request(path, options);
  } catch (err) {
    if (
      err instanceof ApiError &&
      err.status === 401 &&
      onRefresh &&
      !options.skipAuth
    ) {
      const newToken = await onRefresh();
      if (newToken) {
        setAccessToken(newToken);
        return request(path, options);
      }
    }
    throw err;
  }
}
