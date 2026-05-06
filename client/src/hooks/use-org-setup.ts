import { getGithubAppInstallStatus } from "@/lib/api";
import { useQuery } from "@tanstack/react-query";

export function useOrgSetup(orgId: number | undefined) {
  const { data, isLoading } = useQuery({
    queryKey: ["github-app-status", orgId],
    queryFn: () => getGithubAppInstallStatus(orgId!),
    enabled: orgId != null,
    staleTime: 60_000,
  });

  const isSuspended = data?.installed === true && data?.suspended_at != null;
  const installed = data?.installed === true && !isSuspended;

  return {
    installed,
    isSuspended,
    isReady: installed,
    targetLogin: data?.target_login ?? null,
    accountType: data?.account_type ?? null,
    installedByUserName: data?.installed_by_user_name ?? null,
    isLoading,
  };
}
