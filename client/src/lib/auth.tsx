import {
  createContext,
  useContext,
  useState,
  useCallback,
  type ReactNode,
} from 'react';
import { client } from '../client/client.gen';

const API_KEY_KEY = 'github-vote-llm-api-key';

function configureClient(apiKey: string): void {
  client.setConfig({
    baseUrl: window.location.origin,
    headers: { 'X-Api-Key': apiKey },
  });
}

interface AuthContextType {
  apiKey: string | null;
  login: (key: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [apiKey, setApiKey] = useState<string | null>(() => {
    const stored = localStorage.getItem(API_KEY_KEY);
    if (stored) configureClient(stored);
    return stored;
  });

  const login = useCallback((key: string) => {
    localStorage.setItem(API_KEY_KEY, key);
    configureClient(key);
    setApiKey(key);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem(API_KEY_KEY);
    setApiKey(null);
  }, []);

  return (
    <AuthContext.Provider value={{ apiKey, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextType {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
