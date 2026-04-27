import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { CheckCircle2 } from "lucide-react";
import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { SetupShell } from "./SetupShell";

const REDIRECT_DELAY_MS = 2000;

export default function CompletePage() {
  const navigate = useNavigate();

  useEffect(() => {
    const t = setTimeout(() => navigate("/dashboard", { replace: true }), REDIRECT_DELAY_MS);
    return () => clearTimeout(t);
  }, [navigate]);

  return (
    <SetupShell>
      <Card className="px-4 py-4 sm:px-6 sm:py-6 text-center">
        <CardHeader className="gap-3 items-center">
          <div className="w-12 h-12 rounded-full bg-[var(--color-success-muted)] flex items-center justify-center">
            <CheckCircle2 className="w-6 h-6 text-[var(--color-success)]" />
          </div>
          <div className="gap-1.5 flex flex-col">
            <CardTitle>You're all set</CardTitle>
            <CardDescription>
              Your account is now active. Redirecting you to the dashboard…
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent className="pt-2">
          <Button
            variant="ghost"
            className="text-sm"
            onClick={() => navigate("/dashboard", { replace: true })}
          >
            Go to dashboard now
          </Button>
        </CardContent>
      </Card>
    </SetupShell>
  );
}
