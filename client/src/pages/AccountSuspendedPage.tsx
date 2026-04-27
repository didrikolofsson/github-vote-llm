import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useAuth } from "@/lib/auth";
import { ShieldOff } from "lucide-react";
import { SetupShell } from "./setup/SetupShell";

export default function AccountSuspendedPage() {
  const { logout } = useAuth();

  return (
    <SetupShell>
      <Card className="px-4 py-4 sm:px-6 sm:py-6 text-center">
        <CardHeader className="gap-3 items-center">
          <div className="w-12 h-12 rounded-full bg-destructive/10 flex items-center justify-center">
            <ShieldOff className="w-6 h-6 text-destructive" />
          </div>
          <div className="gap-1.5 flex flex-col">
            <CardTitle>Account suspended</CardTitle>
            <CardDescription>
              Your account has been suspended and you no longer have access to
              the dashboard. Please contact support if you believe this is a
              mistake.
            </CardDescription>
          </div>
        </CardHeader>
        <CardContent className="pt-2 flex flex-col gap-2">
          <Button
            variant="outline"
            className="w-full"
            onClick={() => window.open("mailto:support@example.com")}
          >
            Contact support
          </Button>
          <Button
            variant="ghost"
            className="w-full text-sm text-muted-foreground"
            onClick={logout}
          >
            Sign out
          </Button>
        </CardContent>
      </Card>
    </SetupShell>
  );
}
