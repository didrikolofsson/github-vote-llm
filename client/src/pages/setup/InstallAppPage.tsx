import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { GitBranch } from "lucide-react";
import { SetupShell, StepIndicator } from "./SetupShell";

const STEPS = ["Connect GitHub", "Install App"];

export default function InstallAppPage() {
  return (
    <SetupShell>
      <StepIndicator steps={STEPS} currentStep={2} />
      <Card className="px-4 py-4 sm:px-6 sm:py-6">
        <CardHeader className="gap-3">
          <div className="w-10 h-10 rounded-lg bg-muted flex items-center justify-center">
            <GitBranch className="w-5 h-5 text-foreground" />
          </div>
          <div className="gap-1.5 flex flex-col">
            <CardTitle>Install the GitHub App</CardTitle>
            <CardDescription>
              Grant repository access so the AI agent can open pull requests on
              your behalf. This step is disabled until the backend endpoint is
              implemented.
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent className="pt-4 flex flex-col gap-4">
          <div className="rounded-lg border bg-muted/40 px-4 py-3 flex flex-col gap-1.5">
            <p className="text-xs font-medium text-foreground">
              What the app can do
            </p>
            <ul className="text-xs text-muted-foreground space-y-1">
              <li>· Read repository contents and metadata</li>
              <li>· Create branches and open pull requests</li>
              <li>· Post comments on pull requests</li>
            </ul>
          </div>
          <Button className="w-full" disabled>
            Install GitHub App
          </Button>
          <p className="text-xs text-center text-muted-foreground">
            You'll choose which repositories to grant access to on GitHub.
          </p>
        </CardContent>
      </Card>
    </SetupShell>
  );
}
