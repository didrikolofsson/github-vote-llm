import { z } from "zod";
import {
  AuthorizeResponseSchema,
  SignupResponseSchema,
  TokenResponseSchema,
} from "./auth-schemas";
import { request } from "./api-core";

export async function authorize(params: {
  email: string;
  password: string;
  code_challenge: string;
  redirect_uri: string;
}) {
  return request("/auth/authorize", {
    method: "POST",
    body: JSON.stringify(params),
    schema: AuthorizeResponseSchema,
    skipAuth: true,
  });
}

export async function exchangeToken(params: {
  grant_type: "authorization_code";
  code: string;
  code_verifier: string;
  redirect_uri: string;
}) {
  return request("/auth/token", {
    method: "POST",
    body: JSON.stringify(params),
    schema: TokenResponseSchema,
    skipAuth: true,
  });
}

export async function refreshToken(refresh_token: string) {
  return request("/auth/token", {
    method: "POST",
    body: JSON.stringify({ grant_type: "refresh_token", refresh_token }),
    schema: TokenResponseSchema,
    skipAuth: true,
  });
}

export async function revokeToken(refresh_token: string) {
  return request("/auth/revoke", {
    method: "POST",
    body: JSON.stringify({ refresh_token }),
    schema: z.void(),
    skipAuth: true,
  });
}

export async function signup(params: { email: string; password: string }) {
  return request("/users/signup", {
    method: "POST",
    body: JSON.stringify(params),
    schema: SignupResponseSchema,
    skipAuth: true,
  });
}
