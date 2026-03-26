import {
  addEdge,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  Panel,
  ReactFlow,
  useEdgesState,
  useNodesState,
  type Connection,
  type Edge,
  type EdgeChange,
  type Node,
  type NodeChange,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import {
  addFeatureDependency,
  createFeature,
  getRoadmap,
  removeFeatureDependency,
  updateFeaturePosition,
} from "@/lib/api";
import type { Feature, FeatureDependency } from "@/lib/api-schemas";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { FeatureNode, type FeatureNodeData } from "./FeatureNode";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

const NODE_TYPES = { feature: FeatureNode };

const COL_GAP = 300;
const ROW_GAP = 150;

/** Auto-layout features that have no saved position. */
function autoLayout(features: Feature[]): Map<number, { x: number; y: number }> {
  const positioned = new Map<number, { x: number; y: number }>();

  const unpositioned = features.filter(
    (f) => f.roadmap_x == null || f.roadmap_y == null,
  );
  const byArea = new Map<string, Feature[]>();

  for (const f of unpositioned) {
    const area = f.area ?? "__none__";
    if (!byArea.has(area)) byArea.set(area, []);
    byArea.get(area)!.push(f);
  }

  let colIdx = 0;
  for (const [, group] of byArea) {
    group.forEach((f, rowIdx) => {
      positioned.set(f.id, {
        x: colIdx * COL_GAP,
        y: rowIdx * ROW_GAP,
      });
    });
    colIdx++;
  }

  return positioned;
}

function featuresToNodes(
  features: Feature[],
  autoPositions: Map<number, { x: number; y: number }>,
): Node<FeatureNodeData>[] {
  return features.map((f) => {
    const pos =
      f.roadmap_x != null && f.roadmap_y != null
        ? { x: f.roadmap_x, y: f.roadmap_y }
        : (autoPositions.get(f.id) ?? { x: 0, y: 0 });

    return {
      id: String(f.id),
      type: "feature",
      position: pos,
      data: f as FeatureNodeData,
      draggable: !f.roadmap_locked,
    };
  });
}

function depsToEdges(deps: FeatureDependency[]): Edge[] {
  return deps.map((d) => ({
    id: `dep-${d.depends_on}-${d.feature_id}`,
    source: String(d.depends_on),
    target: String(d.feature_id),
    animated: false,
    style: { stroke: "hsl(var(--border))", strokeWidth: 1.5 },
  }));
}

interface AddFeatureDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAdd: (title: string, description: string) => void;
  adding: boolean;
}

function AddFeatureDialog({ open, onOpenChange, onAdd, adding }: AddFeatureDialogProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;
    onAdd(title.trim(), description.trim());
  }

  // Reset on close
  useEffect(() => {
    if (!open) {
      setTitle("");
      setDescription("");
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>New feature</DialogTitle>
            <DialogDescription>
              Add a feature to the roadmap. You can position it by dragging.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-3 my-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="feature-title">Title</Label>
              <Input
                id="feature-title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="e.g. Dark mode support"
                autoFocus
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="feature-desc">Description</Label>
              <Textarea
                id="feature-desc"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Optional description…"
                rows={3}
                className="resize-none"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" size="sm" disabled={!title.trim() || adding}>
              {adding ? "Adding…" : "Add feature"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface RoadmapCanvasProps {
  repoId: number;
}

export function RoadmapCanvas({ repoId }: RoadmapCanvasProps) {
  const queryClient = useQueryClient();
  const [addOpen, setAddOpen] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["repositories", repoId, "roadmap"],
    queryFn: () => getRoadmap(repoId),
    enabled: !!repoId,
  });

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const [nodes, setNodes, onNodesChange] = useNodesState<any>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  // Sync query data into RF state
  useEffect(() => {
    if (!data) return;
    const autoPos = autoLayout(data.features);
    setNodes(featuresToNodes(data.features, autoPos));
    setEdges(depsToEdges(data.dependencies));
  }, [data, setNodes, setEdges]);

  // Save position after drag
  const savePosition = useMutation({
    mutationFn: ({
      featureId,
      x,
      y,
    }: {
      featureId: number;
      x: number;
      y: number;
    }) => updateFeaturePosition(repoId, featureId, x, y, true),
  });

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleNodesChange = useCallback(
    (changes: NodeChange[]) => {
      onNodesChange(changes as any);
    },
    [onNodesChange],
  );

  const handleNodeDragStop = useCallback(
    (_: React.MouseEvent, node: Node) => {
      savePosition.mutate({
        featureId: parseInt(node.id, 10),
        x: node.position.x,
        y: node.position.y,
      });
    },
    [savePosition],
  );

  // Add dependency on connect
  const addDep = useMutation({
    mutationFn: ({ source, target }: { source: string; target: string }) =>
      addFeatureDependency(repoId, parseInt(target, 10), parseInt(source, 10)),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["repositories", repoId, "roadmap"],
      });
    },
  });

  const handleConnect = useCallback(
    (connection: Connection) => {
      if (!connection.source || !connection.target) return;
      setEdges((eds) => addEdge({ ...connection }, eds));
      addDep.mutate({ source: connection.source, target: connection.target });
    },
    [setEdges, addDep],
  );

  // Remove dependency on edge delete
  const removeDep = useMutation({
    mutationFn: ({ featureId, dependsOn }: { featureId: number; dependsOn: number }) =>
      removeFeatureDependency(repoId, featureId, dependsOn),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["repositories", repoId, "roadmap"],
      });
    },
  });

  const handleEdgesChange = useCallback(
    (changes: EdgeChange[]) => {
      for (const change of changes) {
        if (change.type === "remove") {
          const edge = edges.find((e) => e.id === change.id);
          if (edge) {
            removeDep.mutate({
              featureId: parseInt(edge.target, 10),
              dependsOn: parseInt(edge.source, 10),
            });
          }
        }
      }
      onEdgesChange(changes);
    },
    [edges, onEdgesChange, removeDep],
  );

  // Add feature
  const addFeature = useMutation({
    mutationFn: ({ title, description }: { title: string; description: string }) =>
      createFeature(repoId, { title, description: description || undefined }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["repositories", repoId, "roadmap"],
      });
      setAddOpen(false);
    },
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full text-sm text-muted-foreground">
        Loading roadmap…
      </div>
    );
  }

  return (
    <>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={NODE_TYPES}
        onNodesChange={handleNodesChange}
        onEdgesChange={handleEdgesChange}
        onNodeDragStop={handleNodeDragStop}
        onConnect={handleConnect}
        fitView
        fitViewOptions={{ padding: 0.3, maxZoom: 1 }}
        minZoom={0.2}
        maxZoom={2}
        deleteKeyCode="Delete"
        className="bg-muted/20"
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={24}
          size={1}
          className="opacity-40"
        />
        <Controls showInteractive={false} className="[&>button]:border-border" />
        <MiniMap
          nodeStrokeWidth={2}
          nodeColor="hsl(var(--muted-foreground) / 0.3)"
          maskColor="hsl(var(--background) / 0.7)"
          className="border border-border rounded-lg overflow-hidden"
        />
        <Panel position="top-right" className="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            className="bg-background shadow-sm"
            onClick={() => setAddOpen(true)}
          >
            <Plus className="size-3.5" />
            Add feature
          </Button>
        </Panel>

        {nodes.length === 0 && !isLoading && (
          <Panel position="top-center">
            <div className="mt-20 text-center">
              <p className="text-sm font-medium text-muted-foreground">
                No features yet
              </p>
              <p className="text-xs text-muted-foreground/70 mt-1">
                Add your first feature to start building the roadmap.
              </p>
            </div>
          </Panel>
        )}
      </ReactFlow>

      <AddFeatureDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        onAdd={(title, description) => addFeature.mutate({ title, description })}
        adding={addFeature.isPending}
      />
    </>
  );
}
