import { Link } from "react-router-dom";
import { TriangleAlert } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useOrgSetup } from "@/hooks/use-org-setup";

interface SetupBannerProps {
  orgId: number | undefined;
}

export function SetupBanner({ orgId }: SetupBannerProps) {
  const { isReady, isSuspended, isLoading, accountType, targetLogin } =
    useOrgSetup(orgId);

  if (isLoading || isReady) return null;

  const manageURL =
    accountType === "Organization"
      ? `https://github.com/organizations/${targetLogin}/settings/installations`
      : "https://github.com/settings/installations";

  return (
    <Alert variant="warning" className="mb-4">
      <TriangleAlert />
      <AlertDescription>
        {isSuspended ? (
          <>
            The GitHub App is suspended on GitHub. Agent runs are disabled.{" "}
            <a
              href={manageURL}
              target="_blank"
              rel="noopener noreferrer"
              className="font-medium"
            >
              Manage on GitHub →
            </a>
          </>
        ) : (
          <>
            Account setup is incomplete. Install the GitHub App to enable AI
            agent runs.{" "}
            <Link to="/settings" className="font-medium">
              Complete setup →
            </Link>
          </>
        )}
      </AlertDescription>
    </Alert>
  );
}
