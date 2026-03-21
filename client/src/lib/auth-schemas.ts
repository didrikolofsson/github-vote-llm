import { z } from 'zod';

export const AuthorizeResponseSchema = z.object({
  code: z.string(),
  redirect_uri: z.string(),
});

export const TokenResponseSchema = z.object({
  access_token: z.string(),
  refresh_token: z.string().optional(),
  token_type: z.string(),
  expires_in: z.number(),
});

export const SignupResponseSchema = z.object({
  id: z.number(),
  email: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export type AuthorizeResponse = z.infer<typeof AuthorizeResponseSchema>;
export type TokenResponse = z.infer<typeof TokenResponseSchema>;
export type SignupResponse = z.infer<typeof SignupResponseSchema>;
