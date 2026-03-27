import {
  ReactFlow,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  Panel,
  addEdge,
  applyEdgeChanges,
  applyNodeChanges,
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
import { FeatureDrawer } from "./FeatureDrawer";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
  useComboboxAnchor,
} from "@/components/ui/combobox";
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

// Must be defined outside the component — a new object reference on every
// render causes React Flow to remount all nodes.
const NODE_TYPES = { feature: FeatureNode };

const EDGE_STYLE = { stroke: "var(--border)", strokeWidth: 1.5, strokeDasharray: "5 5" };

// ── Layout algorithm ───────────────────────────────────────────────────────────
//
// Assigns each unpositioned feature a column based on its longest dependency
// chain (topological depth), then stacks features within the same column
// vertically, centered around the tallest column.
//
// Tweak these to adjust spacing:
const LAYOUT = {
  nodeWidth: 220,   // px — should match the node's rendered width
  nodeHeight: 80,   // px — approximate rendered height
  columnGap: 80,    // px — horizontal gap between columns
  rowGap: 40,       // px — vertical gap between nodes in the same column
} as const;

function computeLayout(
  features: Feature[],
  dependencies: FeatureDependency[],
): Map<number, { x: number; y: number }> {
  const toLayout = features.filter((f) => f.roadmap_x == null || f.roadmap_y == null);
  if (toLayout.length === 0) return new Map();

  // Build prerequisite map over ALL features so that dependencies on already-positioned
  // features are still taken into account when assigning columns to new ones.
  const allIds = new Set(features.map((f) => f.id));
  const prereqs = new Map<number, number[]>(features.map((f) => [f.id, []]));
  for (const dep of dependencies) {
    if (allIds.has(dep.feature_id) && allIds.has(dep.depends_on)) {
      prereqs.get(dep.feature_id)!.push(dep.depends_on);
    }
  }

  // Column = longest dependency chain to reach this node (memoised DFS).
  // Guard against cycles by tracking the current call stack.
  const colCache = new Map<number, number>();
  function getColumn(id: number, stack = new Set<number>()): number {
    if (colCache.has(id)) return colCache.get(id)!;
    if (stack.has(id)) return 0; // cycle — treat as root
    stack.add(id);
    const pres = prereqs.get(id) ?? [];
    const col =
      pres.length === 0
        ? 0
        : Math.max(...pres.map((p) => getColumn(p, new Set(stack)) + 1));
    colCache.set(id, col);
    return col;
  }
  // Run getColumn on every feature so positioned nodes contribute correct depths.
  for (const f of features) getColumn(f.id);

  // Group by column, sorted by id (creation order) within each column.
  const byColumn = new Map<number, number[]>();
  for (const f of toLayout) {
    const col = colCache.get(f.id)!;
    if (!byColumn.has(col)) byColumn.set(col, []);
    byColumn.get(col)!.push(f.id);
  }
  for (const ids of byColumn.values()) ids.sort((a, b) => a - b);

  // Center shorter columns against the tallest one.
  const colStride = LAYOUT.nodeWidth + LAYOUT.columnGap;
  const rowStride = LAYOUT.nodeHeight + LAYOUT.rowGap;
  const maxRows = Math.max(...[...byColumn.values()].map((ids) => ids.length));

  const positions = new Map<number, { x: number; y: number }>();
  for (const [col, ids] of byColumn) {
    const yOffset = ((maxRows - ids.length) / 2) * rowStride;
    ids.forEach((id, row) => {
      positions.set(id, { x: col * colStride, y: yOffset + row * rowStride });
    });
  }

  return positions;
}

function featuresToNodes(
  features: Feature[],
  autoPositions: Map<number, { x: number; y: number }>,
): Node<FeatureNodeData>[] {
  return features.map((f) => ({
    id: String(f.id),
    type: "feature",
    position:
      f.roadmap_x != null && f.roadmap_y != null
        ? { x: f.roadmap_x, y: f.roadmap_y }
        : (autoPositions.get(f.id) ?? { x: 0, y: 0 }),
    data: f as FeatureNodeData,
    draggable: !f.roadmap_locked,
  }));
}

function depsToEdges(deps: FeatureDependency[]): Edge[] {
  return deps.map((d) => ({
    id: `dep-${d.depends_on}-${d.feature_id}`,
    source: String(d.depends_on),
    target: String(d.feature_id),
    style: EDGE_STYLE,
  }));
}

// ── Add feature dialog ─────────────────────────────────────────────────────────

interface AddFeatureDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAdd: (title: string, description: string, dependsOn: number[]) => void;
  adding: boolean;
  availableFeatures: Feature[];
}

function DepsCombobox({
  features,
  value,
  onValueChange,
}: {
  features: Feature[];
  value: string[];
  onValueChange: (v: string[]) => void;
}) {
  const anchor = useComboboxAnchor();
  const items = features.map((f) => String(f.id));
  const labelOf = (id: string) => features.find((f) => String(f.id) === id)?.title ?? id;

  return (
    <Combobox multiple items={items} value={value} onValueChange={onValueChange}>
      <ComboboxChips ref={anchor} className="min-h-9">
        <ComboboxValue>
          {(values: string[]) => (
            <>
              {values.map((v) => (
                <ComboboxChip key={v}>{labelOf(v)}</ComboboxChip>
              ))}
              <ComboboxChipsInput placeholder={value.length === 0 ? "Search features…" : ""} />
            </>
          )}
        </ComboboxValue>
      </ComboboxChips>
      <ComboboxContent anchor={anchor}>
        <ComboboxEmpty>No features found.</ComboboxEmpty>
        <ComboboxList>
          {items.map((id) => (
            <ComboboxItem key={id} value={id}>
              {labelOf(id)}
            </ComboboxItem>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}

function AddFeatureDialog({
  open,
  onOpenChange,
  onAdd,
  adding,
  availableFeatures,
}: AddFeatureDialogProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [selectedDeps, setSelectedDeps] = useState<string[]>([]);

  useEffect(() => {
    if (!open) {
      setTitle("");
      setDescription("");
      setSelectedDeps([]);
    }
  }, [open]);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;
    onAdd(title.trim(), description.trim(), selectedDeps.map(Number));
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange} modal={false}>
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
            {availableFeatures.length > 0 && (
              <div className="flex flex-col gap-1.5">
                <Label>
                  Depends on
                  <span className="ml-1 font-normal text-muted-foreground">— optional</span>
                </Label>
                <DepsCombobox
                  features={availableFeatures}
                  value={selectedDeps}
                  onValueChange={setSelectedDeps}
                />
                {selectedDeps.length === 0 && (
                  <p className="text-[11px] text-muted-foreground">
                    None selected — will be placed after the latest feature.
                  </p>
                )}
              </div>
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" size="sm" onClick={() => onOpenChange(false)}>
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

// ── Canvas ─────────────────────────────────────────────────────────────────────

interface RoadmapCanvasProps {
  repoId: number;
}

export function RoadmapCanvas({ repoId }: RoadmapCanvasProps) {
  const queryClient = useQueryClient();
  const [addOpen, setAddOpen] = useState(false);
  const [selectedFeature, setSelectedFeature] = useState<Feature | null>(null);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);

  const { data, isLoading } = useQuery({
    queryKey: ["repositories", repoId, "roadmap"],
    queryFn: () => getRoadmap(repoId),
    enabled: !!repoId,
  });

  // Sync nodes + edges whenever server data changes.
  useEffect(() => {
    if (!data) return;
    const positions = computeLayout(data.features, data.dependencies);
    setNodes(featuresToNodes(data.features, positions));
    setEdges(depsToEdges(data.dependencies));
  // setNodes/setEdges are stable React state setters — omitting them is safe.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data]);

  // ── Node / edge change handlers ──────────────────────────────────────────────

  const onNodesChange = useCallback(
    (changes: NodeChange[]) => setNodes((nds) => applyNodeChanges(changes, nds)),
    [],
  );

  const onEdgesChange = useCallback(
    (changes: EdgeChange[]) => {
      // Fire API call for any edge that is being removed.
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
      setEdges((eds) => applyEdgeChanges(changes, eds));
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [edges],
  );

  const onConnect = useCallback(
    (connection: Connection) => {
      if (!connection.source || !connection.target) return;
      // Optimistically add the edge so it appears immediately.
      setEdges((eds) => addEdge({ ...connection, style: EDGE_STYLE }, eds));
      addDep.mutate({ source: connection.source, target: connection.target });
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      const feature = data?.features.find((f) => f.id === parseInt(node.id, 10));
      if (feature) setSelectedFeature(feature);
    },
    [data],
  );

  const onNodeDragStop = useCallback(
    (_: React.MouseEvent, node: Node) => {
      savePosition.mutate({
        featureId: parseInt(node.id, 10),
        x: node.position.x,
        y: node.position.y,
      });
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  // ── Mutations ────────────────────────────────────────────────────────────────

  const savePosition = useMutation({
    mutationFn: ({ featureId, x, y }: { featureId: number; x: number; y: number }) =>
      updateFeaturePosition(repoId, featureId, x, y, true),
  });

  const addDep = useMutation({
    mutationFn: ({ source, target }: { source: string; target: string }) =>
      addFeatureDependency(repoId, parseInt(target, 10), parseInt(source, 10)),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["repositories", repoId, "roadmap"] }),
  });

  const removeDep = useMutation({
    mutationFn: ({ featureId, dependsOn }: { featureId: number; dependsOn: number }) =>
      removeFeatureDependency(repoId, featureId, dependsOn),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["repositories", repoId, "roadmap"] }),
  });

  const addFeature = useMutation({
    mutationFn: async ({
      title,
      description,
      dependsOn,
    }: {
      title: string;
      description: string;
      dependsOn: number[];
    }) => {
      const newFeature = await createFeature(repoId, {
        title,
        description: description || undefined,
      });
      await Promise.all(
        dependsOn.map((depId) => addFeatureDependency(repoId, newFeature.id, depId)),
      );
      return newFeature;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repositories", repoId, "roadmap"] });
      setAddOpen(false);
    },
  });

  // ── Render ───────────────────────────────────────────────────────────────────

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
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeDragStop={onNodeDragStop}
        onNodeClick={onNodeClick}
        onConnect={onConnect}
        fitView
        fitViewOptions={{ padding: 0.3, maxZoom: 1 }}
        minZoom={0.2}
        maxZoom={2}
        deleteKeyCode="Delete"
        className="bg-muted/20"
      >
        <Background variant={BackgroundVariant.Dots} gap={24} size={1} className="opacity-40" />
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
        {nodes.length === 0 && (
          <Panel position="top-center">
            <div className="mt-20 text-center">
              <p className="text-sm font-medium text-muted-foreground">No features yet</p>
              <p className="text-xs text-muted-foreground/70 mt-1">
                Add your first feature to start building the roadmap.
              </p>
            </div>
          </Panel>
        )}
      </ReactFlow>

      <FeatureDrawer
        repoId={repoId}
        feature={selectedFeature}
        onClose={() => setSelectedFeature(null)}
      />

      <AddFeatureDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        availableFeatures={data?.features ?? []}
        onAdd={(title, description, selectedDeps) => {
          // If the user didn't pick any deps, fall back to auto-connecting
          // to the latest existing feature (highest id).
          const dependsOn =
            selectedDeps.length > 0
              ? selectedDeps
              : data?.features.length
                ? [Math.max(...data.features.map((f) => f.id))]
                : [];
          addFeature.mutate({ title, description, dependsOn });
        }}
        adding={addFeature.isPending}
      />
    </>
  );
}
