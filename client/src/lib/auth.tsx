import { decodeJwt } from "jose";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import {
  ApiError,
  signup as apiSignup,
  authorize,
  exchangeToken,
  refreshToken,
  revokeToken,
  setAccessToken,
  setOnRefresh,
} from "./api";
import { TokenPayloadSchema, User, UserSchema } from "./auth-schemas";
import { generateCodeChallenge, generateCodeVerifier } from "./pkce";

const REFRESH_TOKEN_KEY = "github-vote-llm-refresh-token";

function parseUserFromToken(token: string): User | undefined {
  const payload = TokenPayloadSchema.parse(decodeJwt(token));
  return UserSchema.parse({
    id: payload.uid,
    email: payload.email,
  });
}

type AuthContextType = {
  isAuthenticated: boolean;
  isLoading: boolean;
  user: User | undefined;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  error: string | null;
  clearError: () => void;
};

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [accessTokenState, setAccessTokenState] = useState<string | null>(null);
  const [refreshTokenState, setRefreshTokenState] = useState<string | null>(
    () => sessionStorage.getItem(REFRESH_TOKEN_KEY),
  );
  const [isLoading, setIsLoading] = useState(
    !!sessionStorage.getItem(REFRESH_TOKEN_KEY),
  );
  const [error, setError] = useState<string | null>(null);

  const doRefresh = useCallback(async (): Promise<string | null> => {
    const stored = sessionStorage.getItem(REFRESH_TOKEN_KEY);
    if (!stored) return null;
    try {
      const res = await refreshToken(stored);
      setAccessToken(res.access_token);
      setAccessTokenState(res.access_token);
      return res.access_token;
    } catch {
      sessionStorage.removeItem(REFRESH_TOKEN_KEY);
      setRefreshTokenState(null);
      setAccessTokenState(null);
      setAccessToken(null);
      return null;
    }
  }, []);

  useEffect(() => {
    setOnRefresh(doRefresh);
    return () => setOnRefresh(null);
  }, [doRefresh]);

  useEffect(() => {
    if (refreshTokenState && !accessTokenState) {
      doRefresh().finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, [refreshTokenState, accessTokenState, doRefresh]);

  const login = useCallback(async (email: string, password: string) => {
    setError(null);
    try {
      const verifier = await generateCodeVerifier();
      const challenge = await generateCodeChallenge(verifier);
      const redirectUri = window.location.origin;

      const { code } = await authorize({
        email,
        password,
        code_challenge: challenge,
        redirect_uri: redirectUri,
      });

      const tokens = await exchangeToken({
        grant_type: "authorization_code",
        code,
        code_verifier: verifier,
        redirect_uri: redirectUri,
      });

      if (!tokens.access_token) throw new Error("No access token received");
      if (!tokens.refresh_token) throw new Error("No refresh token received");

      setAccessToken(tokens.access_token);
      setAccessTokenState(tokens.access_token);
      sessionStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token);
      setRefreshTokenState(tokens.refresh_token);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setError("Invalid email or password");
      } else {
        setError(err instanceof Error ? err.message : "Login failed");
      }
      throw err;
    }
  }, []);

  const signup = useCallback(
    async (email: string, password: string) => {
      setError(null);
      try {
        await apiSignup({ email, password });
        await login(email, password);
      } catch (err) {
        if (err instanceof ApiError && err.status === 400) {
          const msg =
            typeof err.body === "object" &&
            err.body !== null &&
            "error" in err.body
              ? String((err.body as { error: string }).error)
              : "Signup failed";
          setError(msg);
        } else {
          setError(err instanceof Error ? err.message : "Signup failed");
        }
        throw err;
      }
    },
    [login],
  );

  const logout = useCallback(async () => {
    const stored = sessionStorage.getItem(REFRESH_TOKEN_KEY);
    if (stored) {
      try {
        await revokeToken(stored);
      } catch {
        // Ignore revoke errors
      }
      sessionStorage.removeItem(REFRESH_TOKEN_KEY);
    }
    setRefreshTokenState(null);
    setAccessTokenState(null);
    setAccessToken(null);
    window.history.replaceState({}, "", "/");
  }, []);

  const isAuthenticated = !!accessTokenState;
  const user = accessTokenState
    ? parseUserFromToken(accessTokenState)
    : undefined;

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        isLoading,
        user,
        login,
        signup,
        logout,
        error,
        clearError: () => setError(null),
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextType {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
