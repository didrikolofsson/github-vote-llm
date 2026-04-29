import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from "react";
import { connectGithubAccount, getGitHubInstallUrl } from "./api";

export type GitHubSetupStatus =
  | "inactive"
  | "github_connected"
  | "active"
  | "suspended";

type GitAuthContextType = {
  status: GitHubSetupStatus;
  connectAccount: () => Promise<void>;
  installApp: () => Promise<void>;
};

const GitAuthContext = createContext<GitAuthContextType | null>(null);

export function GitAuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<GitHubSetupStatus>("inactive");

  const connectAccount = useCallback(async () => {
    const authWindow = window.open("about:blank", "_blank");

    try {
      const { authorize_url } = await connectGithubAccount();
      if (!authorize_url) throw new Error("Failed to get authorize URL");

      setStatus("github_connected");
      if (authWindow) {
        authWindow.opener = null;
        authWindow.location.href = authorize_url;
      } else {
        window.location.href = authorize_url;
      }
    } catch (err) {
      authWindow?.close();
      throw err;
    }
  }, []);

  const installApp = useCallback(async () => {
    const { install_url } = await getGitHubInstallUrl();
    window.location.href = install_url;
  }, []);

  return (
    <GitAuthContext.Provider
      value={{
        status,
        connectAccount,
        installApp,
      }}
    >
      {children}
    </GitAuthContext.Provider>
  );
}

export function useGitAuth(): GitAuthContextType {
  const ctx = useContext(GitAuthContext);
  if (!ctx) throw new Error("useGitAuth must be used within GitAuthProvider");
  return ctx;
}
