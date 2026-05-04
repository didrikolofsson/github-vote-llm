import { getGithubAppInstallStatus } from "@/lib/api";
import { useQuery } from "@tanstack/react-query";

export function useOrgSetup(orgId: number | undefined) {
  const { data, isLoading } = useQuery({
    queryKey: ["github-app-status", orgId],
    queryFn: () => getGithubAppInstallStatus(orgId!),
    enabled: orgId != null,
    staleTime: 60_000,
  });

  return {
    isReady: data?.installed === true && !data?.suspended_at,
    isSuspended: data?.suspended_at != null,
    targetLogin: data?.target_login,
    accountType: data?.account_type,
    isLoading,
    status: data,
  };
}
