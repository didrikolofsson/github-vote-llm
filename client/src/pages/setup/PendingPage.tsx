import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Clock } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { SetupShell } from "./SetupShell";

export default function PendingPage() {
  const navigate = useNavigate();

  return (
    <SetupShell>
      <Card className="px-4 py-4 sm:px-6 sm:py-6 text-center">
        <CardHeader className="gap-3 items-center">
          <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center">
            <Clock className="w-6 h-6 text-muted-foreground" />
          </div>
          <div className="gap-1.5 flex flex-col">
            <CardTitle>Waiting for approval</CardTitle>
            <CardDescription>
              Your GitHub App installation is pending approval from an
              organization admin. You'll get access as soon as they approve it.
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent className="pt-2 flex flex-col gap-3">
          <p className="text-xs text-muted-foreground">
            The admin will receive an email from GitHub with the approval
            request. This page will not update automatically — check back after
            you've been notified.
          </p>
          <Button
            variant="outline"
            className="w-full"
            onClick={() => navigate("/setup/install-app")}
          >
            Try installing again
          </Button>
        </CardContent>
      </Card>
    </SetupShell>
  );
}
