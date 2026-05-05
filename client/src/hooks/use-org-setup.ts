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
      isSuspended: data?.suspendedAt != null,
      targetLogin: data?.targetLogin,
      accountType: data?.accountType,
      isLoading,
    };
  }
  return {
    installed: false,
    isSuspended: false,
    targetLogin: null,
    accountType: null,
    isLoading,
  };
}
