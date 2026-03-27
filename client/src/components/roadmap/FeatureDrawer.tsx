import { deleteFeature } from "@/lib/api";
import type { Feature } from "@/lib/api-schemas";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerDescription,
  DrawerFooter,
} from "@/components/ui/drawer";

const STATUS_LABEL: Record<string, string> = {
  open: "Open",
  planned: "Planned",
  in_progress: "In progress",
  done: "Done",
  rejected: "Rejected",
};

interface FeatureDrawerProps {
  repoId: number;
  feature: Feature | null;
  onClose: () => void;
}

export function FeatureDrawer({ repoId, feature, onClose }: FeatureDrawerProps) {
  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: () => deleteFeature(repoId, feature!.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repositories", repoId, "roadmap"] });
      onClose();
    },
  });

  return (
    <Drawer
      open={!!feature}
      onOpenChange={(open) => !open && onClose()}
      direction="right"
    >
      <DrawerContent className="sm:max-w-sm">
        {feature && (
          <>
            <DrawerHeader>
              <DrawerTitle>{feature.title}</DrawerTitle>
              {feature.description && (
                <DrawerDescription>{feature.description}</DrawerDescription>
              )}
            </DrawerHeader>

            <div className="flex flex-col gap-3 px-4 text-sm">
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Status</span>
                <span className="font-medium">
                  {STATUS_LABEL[feature.status] ?? feature.status}
                </span>
              </div>
              {feature.area && (
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Area</span>
                  <span className="font-mono text-xs bg-muted px-1.5 py-0.5 rounded">
                    {feature.area}
                  </span>
                </div>
              )}
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Votes</span>
                <span className="font-medium">{feature.vote_count ?? 0}</span>
              </div>
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
