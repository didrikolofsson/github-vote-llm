import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useOrgSetup } from "@/hooks/use-org-setup";

interface SetupGuardProps {
  orgId: number | undefined;
  children: ReactNode;
}

/**
 * Wraps children and disables them with a tooltip when GitHub App setup is incomplete.
 * Renders children as-is when the org is fully set up.
 */
export function SetupGuard({ orgId, children }: SetupGuardProps) {
  const { isReady, isLoading } = useOrgSetup(orgId);

  if (isLoading || isReady) return <>{children}</>;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex cursor-not-allowed opacity-50">
            <span className="pointer-events-none">{children}</span>
          </span>
        </TooltipTrigger>
        <TooltipContent>
          Install the GitHub App in{" "}
          <Link to="/settings" className="underline">
            Settings
          </Link>{" "}
          to enable agent runs.
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
