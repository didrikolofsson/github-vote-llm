import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { AlertTriangle } from "lucide-react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { SetupShell } from "./SetupShell";

type ErrorReason = "oauth_denied" | "not_admin" | "install_failed" | string;

interface ErrorConfig {
  title: string;
  description: string;
  retryPath: string;
  retryLabel: string;
}

function getErrorConfig(reason: ErrorReason): ErrorConfig {
  switch (reason) {
    case "oauth_denied":
      return {
        title: "GitHub authorization denied",
        description:
          "You declined the GitHub authorization request. We need access to your GitHub account to continue.",
        retryPath: "/setup/connect-github",
        retryLabel: "Try connecting again",
      };
    case "not_admin":
      return {
        title: "Admin access required",
        description:
          "You need to be an owner or admin of the GitHub organization to install the app. Ask an admin to complete the installation.",
        retryPath: "/setup/install-app",
        retryLabel: "Go back to install",
      };
    case "install_failed":
      return {
        title: "Installation failed",
        description:
          "Something went wrong while installing the GitHub App. This is usually a temporary issue — please try again.",
        retryPath: "/setup/install-app",
        retryLabel: "Try installing again",
      };
    default:
      return {
        title: "Something went wrong",
        description:
          "An unexpected error occurred during setup. Please try again or contact support if the problem persists.",
        retryPath: "/setup/connect-github",
        retryLabel: "Start over",
      };
  }
}

export default function ErrorPage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const reason = params.get("reason") ?? "unknown";
  const config = getErrorConfig(reason);

  return (
    <SetupShell>
      <Card className="px-4 py-4 sm:px-6 sm:py-6 text-center">
        <CardHeader className="gap-3 items-center">
          <div className="w-12 h-12 rounded-full bg-destructive/10 flex items-center justify-center">
            <AlertTriangle className="w-6 h-6 text-destructive" />
          </div>
          <div className="gap-1.5 flex flex-col">
            <CardTitle>{config.title}</CardTitle>
            <CardDescription>{config.description}</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="pt-2 flex flex-col gap-2">
          <Button className="w-full" onClick={() => navigate(config.retryPath)}>
            {config.retryLabel}
          </Button>
          <Button
            variant="ghost"
            className="w-full text-sm"
            onClick={() => navigate("/setup/connect-github")}
          >
            Start from the beginning
          </Button>
        </CardContent>
      </Card>
    </SetupShell>
  );
}
