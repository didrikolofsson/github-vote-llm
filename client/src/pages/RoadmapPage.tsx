import { useParams } from "react-router-dom";

export default function RoadmapPage() {
  const { repoId } = useParams<{ repoId: string }>();
  void repoId;

  return (
    <div className="h-[calc(100vh-theme(spacing.12))] -m-8 flex items-center justify-center bg-muted/20">
      <div className="text-center">
        <p className="text-sm font-medium text-muted-foreground">Roadmap coming soon</p>
        <p className="text-xs text-muted-foreground/70 mt-1">
          The interactive feature roadmap will appear here.
        </p>
      </div>
    </div>
  );
}
