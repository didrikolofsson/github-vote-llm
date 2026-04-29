import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useGitAuth } from "@/lib/github-auth";
import { Github } from "lucide-react";
import { SetupShell, StepIndicator } from "./SetupShell";

const STEPS = ["Connect GitHub", "Install App"];

export default function ConnectGitHubPage() {
  const { connectAccount } = useGitAuth();

  return (
    <SetupShell>
      <StepIndicator steps={STEPS} currentStep={1} />
      <Card className="px-4 py-4 sm:px-6 sm:py-6">
        <CardHeader className="gap-3">
          <div className="w-10 h-10 rounded-lg bg-muted flex items-center justify-center">
            <Github className="w-5 h-5 text-foreground" />
          </div>
          <div className="gap-1.5 flex flex-col">
            <CardTitle>Connect your GitHub account</CardTitle>
            <CardDescription>
              We need to link your GitHub identity before installing the app.
              This lets us reliably match any installation event back to your
              organization — even if something goes wrong mid-flow.
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent className="pt-4 flex flex-col gap-3">
          <Button className="w-full" onClick={connectAccount}>
            <Github className="w-4 h-4 mr-2" />
            Connect GitHub
          </Button>
          <p className="text-xs text-center text-muted-foreground">
            GitHub authorization opens in a new tab.
          </p>
        </CardContent>
      </Card>
    </SetupShell>
  );
}
