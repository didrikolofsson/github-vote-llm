import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { CheckCircle2 } from "lucide-react";
import { useEffect, useMemo } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { SetupShell } from "./SetupShell";

type PopupKind = "oauth" | "app_install" | "app_update";

function parseKind(v: string | null): PopupKind | null {
  if (v === "oauth" || v === "app_install" || v === "app_update") return v;
  return null;
}

export default function PopupCompletePage() {
  const [params] = useSearchParams();

  const payload = useMemo(() => {
    const kind = parseKind(params.get("kind"));
    const ok = params.get("ok") === "1";
    const orgIdRaw = params.get("org_id");
    const org_id =
      orgIdRaw && /^\d+$/.test(orgIdRaw) ? Number(orgIdRaw) : undefined;

    return { kind, ok, org_id };
  }, [params]);

  useEffect(() => {
    if (!payload.kind) return;

    if (window.opener && payload.ok) {
      window.opener.postMessage(
        { type: "github:complete", ...payload },
        window.location.origin,
      );
      window.close();
    }
  }, [payload]);

  const title = payload.ok ? "Done" : "Something went wrong";
  const description = payload.ok
    ? "You can close this window and continue."
    : "Please return to the app and try again.";

  return (
    <SetupShell>
      <Card className="px-4 py-4 sm:px-6 sm:py-6 text-center">
        <CardHeader className="gap-3 items-center">
          <div className="w-12 h-12 rounded-full bg-(--color-success-muted) flex items-center justify-center">
            <CheckCircle2 className="w-6 h-6 text-(--color-success)" />
          </div>
          <div className="gap-1.5 flex flex-col">
            <CardTitle>{title}</CardTitle>
            <CardDescription>{description}</CardDescription>
          </div>
        </CardHeader>
        <CardContent className="pt-2 flex flex-col gap-2">
          <Button variant="outline" asChild>
            <Link to="/settings">Back to settings</Link>
          </Button>
          <Button variant="ghost" asChild>
            <Link to="/dashboard">Back to dashboard</Link>
          </Button>
        </CardContent>
      </Card>
    </SetupShell>
  );
}
