import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { createPortalComment } from "@/lib/portal-api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";

interface CommentFormProps {
  orgSlug: string;
  repoName: string;
  featureId: number;
  onCommentAdded: () => void;
}

export function CommentForm({ orgSlug, repoName, featureId, onCommentAdded }: CommentFormProps) {
  const [body, setBody] = useState("");
  const [authorName, setAuthorName] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      createPortalComment(orgSlug, repoName, featureId, body.trim(), authorName.trim() || undefined),
    onSuccess: () => {
      setBody("");
      onCommentAdded();
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!body.trim()) return;
    mutation.mutate();
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="comment-author" className="text-xs text-muted-foreground">
          Name <span className="text-muted-foreground/60">(optional)</span>
        </Label>
        <Input
          id="comment-author"
          value={authorName}
          onChange={(e) => setAuthorName(e.target.value)}
          placeholder="Anonymous"
          className="h-8 text-sm"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="comment-body" className="text-xs text-muted-foreground">
          Comment
        </Label>
        <Textarea
          id="comment-body"
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="Share your thoughts…"
          rows={3}
          className="text-sm resize-none"
        />
      </div>
      {mutation.isError && (
        <p className="text-xs text-destructive">
          {mutation.error instanceof Error ? mutation.error.message : "Failed to post comment"}
        </p>
      )}
      <Button
        type="submit"
        size="sm"
        disabled={!body.trim() || mutation.isPending}
        className="self-end"
      >
        {mutation.isPending ? "Posting…" : "Post comment"}
      </Button>
    </form>
  );
}
