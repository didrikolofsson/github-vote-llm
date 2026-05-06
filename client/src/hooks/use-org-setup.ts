import { getGithubAppInstallStatus } from "@/lib/api";
import { useQuery } from "@tanstack/react-query";

export function useOrgSetup(orgId: number | undefined) {
  const { data, isLoading } = useQuery({
    queryKey: ["github-app-status", orgId],
    queryFn: () => getGithubAppInstallStatus(orgId!),
    enabled: orgId != null,
    staleTime: 60_000,
  });

  if (data?.installed) {
    return {
      installed: true,
      isSuspended: data?.suspended_at != null,
      targetLogin: data?.target_login,
      accountType: data?.account_type,
      installedByUserName: data?.installed_by_user_name,
      isLoading,
    };
  }
  return {
    installed: false,
    isSuspended: false,
    targetLogin: null,
    accountType: null,
    installedByUserName: null,
    isLoading,
  };
}
