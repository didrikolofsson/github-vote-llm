import {
  ReactFlow,
  Background,
  BackgroundVariant,
  Panel,
  addEdge,
  applyEdgeChanges,
  applyNodeChanges,
  useReactFlow,
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
import ELK from "elkjs/lib/elk.bundled.js";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { LayoutGrid, Maximize2, Plus, ZoomIn, ZoomOut } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
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

const EDGE_STYLE = {
  stroke: "var(--border)",
  strokeWidth: 1.5,
  strokeDasharray: "5 5",
};

// ── Layout ─────────────────────────────────────────────────────────────────────

const NODE_WIDTH = 220;
const NODE_HEIGHT = 110;

const elk = new ELK();

// Uses ELK's layered algorithm to place unpositioned features.
// Already-positioned features have their x/y passed as hints so ELK uses them
// for layer assignment (INTERACTIVE strategy), keeping new nodes contextually
// placed relative to the existing layout.
// Only positions for previously-unpositioned nodes are returned; existing
// positions are never overwritten here.
async function computeLayout(
  features: Feature[],
  dependencies: FeatureDependency[],
): Promise<Map<number, { x: number; y: number }>> {
  const unpositioned = features.filter(
    (f) => f.roadmap_x == null || f.roadmap_y == null,
  );
  if (unpositioned.length === 0) return new Map();

  const graph = {
    id: "root",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": "RIGHT",
      // INTERACTIVE strategies read the provided x/y of already-placed nodes
      // to infer their layer + ordering, so new nodes land in the right column.
      "elk.layered.layering.strategy": "INTERACTIVE",
      "elk.layered.crossingMinimization.semiInteractive": "true",
      "elk.spacing.nodeNode": "40",
      "elk.layered.spacing.nodeNodeBetweenLayers": "80",
    },
    children: features.map((f) => ({
      id: String(f.id),
      width: NODE_WIDTH,
      height: NODE_HEIGHT,
      ...(f.roadmap_x != null && f.roadmap_y != null
        ? { x: f.roadmap_x, y: f.roadmap_y }
        : {}),
    })),
    edges: dependencies.map((d) => ({
      id: `dep-${d.depends_on}-${d.feature_id}`,
      sources: [String(d.depends_on)],
      targets: [String(d.feature_id)],
    })),
  };

  const result = await elk.layout(graph);

  const unpositionedIds = new Set(unpositioned.map((f) => f.id));

  // Build a map of direct prerequisites per feature.
  const prereqMap = new Map<number, number[]>();
  for (const d of dependencies) {
    if (!prereqMap.has(d.feature_id)) prereqMap.set(d.feature_id, []);
    prereqMap.get(d.feature_id)!.push(d.depends_on);
  }

  // Index positioned features by id for quick lookup.
  const positionedById = new Map<number, { x: number; y: number }>();
  for (const f of features) {
    if (f.roadmap_x != null && f.roadmap_y != null) {
      positionedById.set(f.id, { x: f.roadmap_x, y: f.roadmap_y });
    }
  }

  // ELK gives us the correct x (layer/column) for each new node. But its y is
  // computed in its own clean layout space, divorced from where the user has
  // actually placed dependencies. Override y with the average y of the node's
  // direct prerequisites that have saved positions — this aligns new nodes with
  // their actual parents on the canvas. Fall back to ELK's y when no positioned
  // prerequisites exist (e.g. initial full layout).
  const elkPositions = new Map<number, { x: number; y: number }>();
  for (const node of result.children ?? []) {
    const id = parseInt(node.id, 10);
    if (!unpositionedIds.has(id) || node.x == null || node.y == null) continue;

    const prereqs = prereqMap.get(id) ?? [];
    const depYs = prereqs
      .map((depId) => positionedById.get(depId)?.y)
      .filter((y): y is number => y != null);

    const idealY =
      depYs.length > 0
        ? depYs.reduce((sum, y) => sum + y, 0) / depYs.length
        : node.y;

    elkPositions.set(id, { x: node.x, y: idealY });
  }

  // Resolve overlaps: for each new node, try y offsets (±stride) until the
  // bounding box doesn't collide with any existing or already-resolved node.
  const ROW_STRIDE = NODE_HEIGHT + 40;
  function overlaps(a: { x: number; y: number }, b: { x: number; y: number }) {
    return (
      Math.abs(a.x - b.x) < NODE_WIDTH + 10 &&
      Math.abs(a.y - b.y) < NODE_HEIGHT + 10
    );
  }

  const occupied: { x: number; y: number }[] = [...positionedById.values()];

  const positions = new Map<number, { x: number; y: number }>();
  for (const [id, pos] of elkPositions) {
    const allOccupied = [...occupied, ...positions.values()];
    let finalPos = pos;
    for (let i = 0; i < 50; i++) {
      // Sequence: 0, +stride, -stride, +2*stride, -2*stride, …
      const offset =
        i === 0 ? 0 : Math.ceil(i / 2) * ROW_STRIDE * (i % 2 === 1 ? 1 : -1);
      const candidate = { x: pos.x, y: pos.y + offset };
      if (!allOccupied.some((o) => overlaps(o, candidate))) {
        finalPos = candidate;
        break;
      }
    }
    occupied.push(finalPos);
    positions.set(id, finalPos);
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
  }));
}

function depsToEdges(deps: FeatureDependency[]): Edge[] {
  return deps.map((d) => ({
    id: `dep-${d.depends_on}-${d.feature_id}`,
    source: String(d.depends_on),
    target: String(d.feature_id),
    style: EDGE_STYLE,
    animated: true,
  }));
}

// ── Canvas toolbar ─────────────────────────────────────────────────────────────

function CanvasToolbar({
  onAdd,
  onResetLayout,
  resetting,
}: {
  onAdd: () => void;
  onResetLayout: () => void;
  resetting: boolean;
}) {
  const { zoomIn, zoomOut, fitView } = useReactFlow();

  return (
    <Panel position="top-center">
      <TooltipProvider>
        <div className="flex flex-row items-center gap-1 bg-background border border-border rounded-xl shadow-sm p-1.5">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                size="icon"
                variant="default"
                className="size-8"
                onClick={onAdd}
              >
                <Plus data-icon />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">Add feature</TooltipContent>
          </Tooltip>

          <Separator orientation="vertical" className="h-5 mx-0.5 my-auto" />

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                size="icon"
                variant="ghost"
                className="size-8 text-muted-foreground"
                onClick={onResetLayout}
                disabled={resetting}
              >
                <LayoutGrid data-icon />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">Reset layout</TooltipContent>
          </Tooltip>

          <Separator orientation="vertical" className="h-5 mx-0.5 my-auto" />

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                size="icon"
                variant="ghost"
                className="size-8 text-muted-foreground"
                onClick={() => zoomIn()}
              >
                <ZoomIn data-icon />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">Zoom in</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                size="icon"
                variant="ghost"
                className="size-8 text-muted-foreground"
                onClick={() => zoomOut()}
              >
                <ZoomOut data-icon />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">Zoom out</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                size="icon"
                variant="ghost"
                className="size-8 text-muted-foreground"
                onClick={() => fitView({ padding: 0.3, maxZoom: 1 })}
              >
                <Maximize2 data-icon />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">Fit view</TooltipContent>
          </Tooltip>
        </div>
      </TooltipProvider>
    </Panel>
  );
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
  container,
}: {
  features: Feature[];
  value: string[];
  onValueChange: (v: string[]) => void;
  container?: HTMLElement | null;
}) {
  const anchor = useComboboxAnchor();
  const items = features.map((f) => String(f.id));
  const labelOf = (id: string) =>
    features.find((f) => String(f.id) === id)?.title ?? id;

  return (
    <Combobox
      multiple
      items={items}
      value={value}
      onValueChange={onValueChange}
    >
      <ComboboxChips ref={anchor} className="min-h-9">
        <ComboboxValue>
          {(values: string[]) => (
            <>
              {values.map((v) => (
                <ComboboxChip key={v}>{labelOf(v)}</ComboboxChip>
              ))}
              <ComboboxChipsInput
                placeholder={value.length === 0 ? "Search features…" : ""}
              />
            </>
          )}
        </ComboboxValue>
      </ComboboxChips>
      <ComboboxContent anchor={anchor} container={container}>
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
  const [container, setContainer] = useState<HTMLDivElement | null>(null);

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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <div ref={setContainer} />
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
                  <span className="ml-1 font-normal text-muted-foreground">
                    — optional
                  </span>
                </Label>
                <DepsCombobox
                  features={availableFeatures}
                  value={selectedDeps}
                  onValueChange={setSelectedDeps}
                  container={container}
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

// ── Canvas ─────────────────────────────────────────────────────────────────────

interface RoadmapCanvasProps {
  repoId: number;
}

export function RoadmapCanvas({ repoId }: RoadmapCanvasProps) {
  const queryClient = useQueryClient();
  const [addOpen, setAddOpen] = useState(false);
  const [selectedFeatureId, setSelectedFeatureId] = useState<number | null>(
    null,
  );
  const [nodes, setNodes] = useState<Node[]>([]);
  const [edges, setEdges] = useState<Edge[]>([]);

  const { data, isLoading } = useQuery({
    queryKey: ["repositories", repoId, "roadmap"],
    queryFn: () => getRoadmap(repoId),
    enabled: !!repoId,
  });

  // Sync nodes + edges whenever server data changes.
  // computeLayout is async (ELK), so we use .then() inside the effect.
  // After layout, persist ELK-computed positions so they survive navigation.
  useEffect(() => {
    if (!data) return;
    computeLayout(data.features, data.dependencies).then((positions) => {
      setNodes(featuresToNodes(data.features, positions));
      setEdges(depsToEdges(data.dependencies));
      for (const [featureId, { x, y }] of positions) {
        savePosition.mutate({ featureId, x, y });
      }
    });
    // setNodes/setEdges/savePosition are stable references — omitting them is safe.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data]);

  // ── Node / edge change handlers ──────────────────────────────────────────────

  const onNodesChange = useCallback(
    (changes: NodeChange[]) =>
      setNodes((nds) => applyNodeChanges(changes, nds)),
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

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    setSelectedFeatureId(parseInt(node.id, 10));
  }, []);

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

  const addDep = useMutation({
    mutationFn: ({ source, target }: { source: string; target: string }) =>
      addFeatureDependency(repoId, parseInt(target, 10), parseInt(source, 10)),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: ["repositories", repoId, "roadmap"],
      }),
  });

  const removeDep = useMutation({
    mutationFn: ({
      featureId,
      dependsOn,
    }: {
      featureId: number;
      dependsOn: number;
    }) => removeFeatureDependency(repoId, featureId, dependsOn),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: ["repositories", repoId, "roadmap"],
      }),
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
        dependsOn.map((depId) =>
          addFeatureDependency(repoId, newFeature.id, depId),
        ),
      );
      return newFeature;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["repositories", repoId, "roadmap"],
      });
      queryClient.invalidateQueries({
        queryKey: ["repository", repoId, "meta"],
      });
      setAddOpen(false);
    },
  });

  // ── Reset layout ─────────────────────────────────────────────────────────────

  const [resetting, setResetting] = useState(false);

  async function handleResetLayout() {
    if (!data || resetting) return;
    setResetting(true);
    // Treat all features as unpositioned so ELK recomputes every node.
    const unpositioned = data.features.map((f) => ({
      ...f,
      roadmap_x: null,
      roadmap_y: null,
    }));
    const positions = await computeLayout(unpositioned, data.dependencies);
    setNodes(featuresToNodes(unpositioned, positions));
    for (const [featureId, { x, y }] of positions) {
      await savePosition.mutateAsync({ featureId, x, y });
    }
    setResetting(false);
  }

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
        <Background variant={BackgroundVariant.Dots} gap={24} size={1} />
        {nodes.length === 0 && (
          <div className="absolute top-0 left-0 right-0 bottom-0 flex items-center justify-center">
            <div className="relative text-center">
              <p className="text-sm font-medium text-muted-foreground">
                No features yet
              </p>
              <p className="text-xs text-muted-foreground/70 mt-1">
                Add your first feature to start building the roadmap.
              </p>
            </div>
          </div>
        )}
        <CanvasToolbar
          onAdd={() => setAddOpen(true)}
          onResetLayout={handleResetLayout}
          resetting={resetting}
        />
      </ReactFlow>

      <FeatureDrawer
        repoId={repoId}
        feature={data?.features.find((f) => f.id === selectedFeatureId) ?? null}
        allFeatures={data?.features ?? []}
        dependencies={data?.dependencies ?? []}
        onClose={() => setSelectedFeatureId(null)}
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
