import { CheckCircle2 } from "lucide-react";
import type { PortalFeature } from "@/lib/portal-api";

interface RecentlyShippedProps {
  features: PortalFeature[];
  onSelect: (featureId: number) => void;
}

export function RecentlyShipped({ features, onSelect }: RecentlyShippedProps) {
  if (features.length === 0) return null;

  return (
    <section>
      <div className="mb-4">
        <h2 className="text-lg font-semibold tracking-tight">Recently shipped</h2>
        <p className="text-sm text-muted-foreground mt-0.5">The latest features we have delivered.</p>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        {features.map((f) => (
          <button
            key={f.id}
            type="button"
            className="text-left rounded-xl border border-border bg-card hover:bg-accent/50 transition-colors px-4 py-3"
            onClick={() => onSelect(f.id)}
          >
            <div className="flex items-start gap-2.5">
              <CheckCircle2 className="size-4 shrink-0 text-emerald-400 mt-0.5" />
              <div className="min-w-0">
                <p className="text-sm font-medium text-foreground leading-snug line-clamp-2">
                  {f.title}
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  {new Date(f.updated_at).toLocaleDateString("en-US", {
                    month: "short",
                    day: "numeric",
                    year: "numeric",
                  })}
                </p>
              </div>
            </div>
          </button>
        ))}
      </div>
    </section>
  );
}
