import type { PortalFeature } from "@/lib/portal-api";
import { FeatureCard } from "./FeatureCard";

interface RoadmapColumnsProps {
  planned: PortalFeature[];
  inProgress: PortalFeature[];
  done: PortalFeature[];
  onSelect: (featureId: number) => void;
}

interface ColumnProps {
  title: string;
  dot: string;
  features: PortalFeature[];
  onSelect: (featureId: number) => void;
}

function Column({ title, dot, features, onSelect }: ColumnProps) {
  return (
    <div className="flex flex-col gap-3 min-w-0">
      <div className="flex items-center gap-2">
        <span className={`size-2 rounded-full ${dot}`} />
        <h3 className="text-sm font-semibold text-foreground">{title}</h3>
        <span className="text-xs text-muted-foreground ml-auto">{features.length}</span>
      </div>
      {features.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border py-8 text-center">
          <p className="text-xs text-muted-foreground">Nothing here yet.</p>
        </div>
      ) : (
        <div className="flex flex-col gap-2">
          {features.map((f) => (
            <FeatureCard key={f.id} feature={f} onClick={onSelect} compact />
          ))}
        </div>
      )}
    </div>
  );
}

export function RoadmapColumns({ planned, inProgress, done, onSelect }: RoadmapColumnsProps) {
  return (
    <section>
      <div className="mb-4">
        <h2 className="text-lg font-semibold tracking-tight">Roadmap</h2>
        <p className="text-sm text-muted-foreground mt-0.5">What we are working on.</p>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Column title="Planned" dot="bg-violet-400" features={planned} onSelect={onSelect} />
        <Column title="In Progress" dot="bg-amber-400" features={inProgress} onSelect={onSelect} />
        <Column title="Done" dot="bg-emerald-400" features={done} onSelect={onSelect} />
      </div>
    </section>
  );
}
