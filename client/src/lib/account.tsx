import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from "react";

export type AccountStatus =
  | "inactive"
  | "github_connected"
  | "active"
  | "suspended";

type AccountContextType = {
  status: AccountStatus;
  github_account_login: string | null;
  connectGitHub: () => void;
  installApp: () => void;
  reset: () => void;
  setStatus: (s: AccountStatus) => void;
};

const AccountContext = createContext<AccountContextType | null>(null);

const MOCK_GITHUB_LOGIN = "acme-corp";

export function AccountProvider({ children }: { children: ReactNode }) {
  const [status, setStatusState] = useState<AccountStatus>("inactive");
  const [github_account_login, setGithubAccountLogin] = useState<
    string | null
  >(null);

  const connectGitHub = useCallback(() => {
    setStatusState("github_connected");
    setGithubAccountLogin(MOCK_GITHUB_LOGIN);
  }, []);

  const installApp = useCallback(() => {
    setStatusState("active");
  }, []);

  const reset = useCallback(() => {
    setStatusState("inactive");
    setGithubAccountLogin(null);
  }, []);

  const setStatus = useCallback((s: AccountStatus) => {
    setStatusState(s);
    if (s === "github_connected" || s === "active") {
      setGithubAccountLogin(MOCK_GITHUB_LOGIN);
    } else {
      setGithubAccountLogin(null);
    }
  }, []);

  return (
    <AccountContext.Provider
      value={{
        status,
        github_account_login,
        connectGitHub,
        installApp,
        reset,
        setStatus,
      }}
    >
      {children}
    </AccountContext.Provider>
  );
}

export function useAccount(): AccountContextType {
  const ctx = useContext(AccountContext);
  if (!ctx) throw new Error("useAccount must be used within AccountProvider");
  return ctx;
}
