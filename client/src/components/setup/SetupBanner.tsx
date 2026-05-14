import { Link } from "react-router-dom";
import { TriangleAlert } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useOrgSetup } from "@/hooks/use-org-setup";

interface SetupBannerProps {
  orgId: number | undefined;
}

export function SetupBanner({ orgId }: SetupBannerProps) {
  const { isReady, isSuspended, isLoading } = useOrgSetup(orgId);

  if (isLoading || isReady) return null;

  return (
    <Alert variant="warning">
      <TriangleAlert />
      <AlertDescription>
        {isSuspended ? (
          <>
            The GitHub App is suspended on GitHub. Agent runs are disabled.{" "}
            <Link to="/settings" className="font-medium">
              Manage in settings →
            </Link>
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
