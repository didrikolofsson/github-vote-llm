import type { PortalFeature } from "@/lib/portal-api";
import { FeatureCard } from "./FeatureCard";

interface ProposalsBoardProps {
  proposals: PortalFeature[];
  onVote: (featureId: number) => void;
  onSelect: (featureId: number) => void;
}

export function ProposalsBoard({ proposals, onVote, onSelect }: ProposalsBoardProps) {
  return (
    <section>
      <div className="mb-4">
        <h2 className="text-lg font-semibold tracking-tight">Feature requests</h2>
        <p className="text-sm text-muted-foreground mt-0.5">
          Vote on what matters most to you.
        </p>
      </div>

      {proposals.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border py-12 text-center">
          <p className="text-sm text-muted-foreground">No open proposals yet.</p>
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {proposals.map((f) => (
            <FeatureCard
              key={f.id}
              feature={f}
              onVote={onVote}
              onClick={onSelect}
            />
          ))}
        </div>
      )}
    </section>
  );
}
