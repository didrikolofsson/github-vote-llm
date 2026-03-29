import {
  addFeatureDependency,
  deleteFeature,
  removeFeatureDependency,
  updateFeature,
} from "@/lib/api";
import type {
  Feature,
  FeatureDependency,
  FeatureStatus,
} from "@/lib/api-schemas";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Trash2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerFooter,
  DrawerTitle,
} from "@/components/ui/drawer";
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

const STATUS_OPTIONS: { value: FeatureStatus; label: string }[] = [
  { value: "open", label: "Open" },
  { value: "planned", label: "Planned" },
  { value: "in_progress", label: "In progress" },
  { value: "done", label: "Done" },
  { value: "rejected", label: "Rejected" },
];

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

interface FeatureDrawerProps {
  repoId: number;
  feature: Feature | null;
  allFeatures: Feature[];
  dependencies: FeatureDependency[];
  onClose: () => void;
}

export function FeatureDrawer({
  repoId,
  feature,
  allFeatures,
  dependencies,
  onClose,
}: FeatureDrawerProps) {
  const queryClient = useQueryClient();
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [container, setContainer] = useState<HTMLDivElement | null>(null);

  // Local edit state
  const [title, setTitle] = useState("");
  const [titleDraft, setTitleDraft] = useState("");
  const [description, setDescription] = useState("");
  const [descriptionDraft, setDescriptionDraft] = useState("");

  useEffect(() => {
    if (feature) {
      setTitle(feature.title);
      setTitleDraft(feature.title);
      setDescription(feature.description ?? "");
      setDescriptionDraft(feature.description ?? "");
    }
  }, [feature?.id, feature?.title, feature?.description]);

  // Current dependency IDs for this feature (features it depends on)
  const currentDepIds = dependencies
    .filter((d) => d.feature_id === feature?.id)
    .map((d) => String(d.depends_on));

  // Features available as dependency options (all except the current feature)
  const depOptions = allFeatures.filter((f) => f.id !== feature?.id);

  function invalidate() {
    queryClient.invalidateQueries({
      queryKey: ["repositories", repoId, "roadmap"],
    });
    queryClient.invalidateQueries({
      queryKey: ["repository", repoId, "meta"],
    });
  }

  const patch = useMutation({
    mutationFn: (p: {
      title?: string;
      description?: string;
      status?: string;
    }) => updateFeature(repoId, feature!.id, p),
    onSuccess: invalidate,
  });

  const addDep = useMutation({
    mutationFn: (dependsOn: number) =>
      addFeatureDependency(repoId, feature!.id, dependsOn),
    onSuccess: invalidate,
  });

  const removeDep = useMutation({
    mutationFn: (dependsOn: number) =>
      removeFeatureDependency(repoId, feature!.id, dependsOn),
    onSuccess: invalidate,
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteFeature(repoId, feature!.id),
    onSuccess: () => {
      invalidate();
      onClose();
    },
  });

  function handleDescriptionBlur() {
    if (descriptionDraft !== description) {
      patch.mutate({ description: descriptionDraft });
    }
  }

  function handleTitleBlur() {
    const trimmed = titleDraft.trim();
    if (trimmed && trimmed !== title) {
      patch.mutate({ title: trimmed });
    } else {
      setTitleDraft(title);
    }
  }

  function handleTitleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") (e.target as HTMLInputElement).blur();
    if (e.key === "Escape") {
      setTitleDraft(title);
      (e.target as HTMLInputElement).blur();
    }
  }

  function handleDepsChange(newIds: string[]) {
    const prev = new Set(currentDepIds);
    const next = new Set(newIds);

    for (const id of next) {
      if (!prev.has(id)) addDep.mutate(Number(id));
    }
    for (const id of prev) {
      if (!next.has(id)) removeDep.mutate(Number(id));
    }
  }

  return (
    <Drawer
      open={!!feature}
      onOpenChange={(open) => !open && onClose()}
      direction="right"
    >
      <DrawerContent
        aria-describedby={undefined}
        className="w-[420px] sm:max-w-[420px]"
      >
        {feature && (
          <>
            <div
              ref={(el) => {
                containerRef.current = el;
                setContainer(el);
              }}
            />
            <DrawerHeader className="pb-2">
              <DrawerTitle className="sr-only">{title}</DrawerTitle>
              <Input
                value={titleDraft}
                onChange={(e) => setTitleDraft(e.target.value)}
                onBlur={handleTitleBlur}
                onKeyDown={handleTitleKeyDown}
                className="text-base font-semibold border-transparent shadow-none px-0 focus-visible:border-input focus-visible:shadow-sm focus-visible:px-3 transition-all"
              />
            </DrawerHeader>

            <div className="flex flex-col gap-5 px-4 py-2">
              <div className="flex flex-col gap-1.5">
                <Label className="text-muted-foreground text-xs">
                  Description
                </Label>
                <Textarea
                  value={descriptionDraft}
                  onChange={(e) => setDescriptionDraft(e.target.value)}
                  onBlur={handleDescriptionBlur}
                  placeholder="Add a description…"
                  rows={4}
                  className="resize-none text-sm"
                />
              </div>

              <div className="flex flex-col gap-1.5">
                <Label className="text-muted-foreground text-xs">Status</Label>
                <Select
                  value={feature.status}
                  onValueChange={(v) => patch.mutate({ status: v })}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {STATUS_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {depOptions.length > 0 && (
                <div className="flex flex-col gap-1.5">
                  <Label className="text-muted-foreground text-xs">
                    Depends on
                  </Label>
                  <DepsCombobox
                    features={depOptions}
                    value={currentDepIds}
                    onValueChange={handleDepsChange}
                    container={container}
                  />
                </div>
              )}
            </div>

            <DrawerFooter>
              <Button
                variant="destructive"
                size="sm"
                className="w-full"
                onClick={() => deleteMutation.mutate()}
                disabled={deleteMutation.isPending}
              >
                <Trash2 data-icon="inline-start" />
                {deleteMutation.isPending ? "Deleting…" : "Delete feature"}
              </Button>
            </DrawerFooter>
          </>
        )}
      </DrawerContent>
    </Drawer>
  );
}
