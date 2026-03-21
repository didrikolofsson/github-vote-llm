import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listRepos,
  updateRepoConfig,
  deleteRepoConfig,
  type RepoConfig,
  type UpdateRepoConfigRequest,
} from '@/lib/api';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';

// ─── Shared styles ───────────────────────────────────────────────────────────

const SECTION_LABEL_CLASS =
  'text-[9px] tracking-[0.25em] uppercase text-muted-foreground mb-3';

const FIELD_LABEL_CLASS =
  'block text-[10px] tracking-[0.15em] uppercase text-muted-foreground mb-1.5';

// ─── Defaults ─────────────────────────────────────────────────────────────────

const DEFAULT_FORM: UpdateRepoConfigRequest = {
  label_approved: 'approved-for-dev',
  label_in_progress: 'llm-in-progress',
  label_done: 'llm-pr-created',
  label_failed: 'llm-failed',
  label_feature_request: 'feature-request',
  vote_threshold: 3,
  timeout_minutes: 30,
  max_budget_usd: 5,
  is_board_public: false,
};

// ─── Shared form fields renderer ──────────────────────────────────────────────

function FormFields({
  form,
  setForm,
  showOwnerRepo,
  owner,
  repo,
  setOwner,
  setRepo,
}: {
  form: UpdateRepoConfigRequest;
  setForm: React.Dispatch<React.SetStateAction<UpdateRepoConfigRequest>>;
  showOwnerRepo?: boolean;
  owner?: string;
  repo?: string;
  setOwner?: (v: string) => void;
  setRepo?: (v: string) => void;
}) {
  function textField(label: string, key: keyof UpdateRepoConfigRequest) {
    return (
      <div key={key}>
        <Label className={FIELD_LABEL_CLASS}>{label}</Label>
        <Input
          type="text"
          value={String(form[key] ?? '')}
          onChange={(e) => setForm((f) => ({ ...f, [key]: e.target.value }))}
          className="text-xs tracking-[0.03em] rounded-none h-8"
        />
      </div>
    );
  }

  function numField(label: string, key: keyof UpdateRepoConfigRequest) {
    return (
      <div key={key}>
        <Label className={FIELD_LABEL_CLASS}>{label}</Label>
        <Input
          type="number"
          value={String(form[key] ?? '')}
          onChange={(e) => setForm((f) => ({ ...f, [key]: Number(e.target.value) }))}
          className="text-xs tracking-[0.03em] rounded-none h-8"
        />
      </div>
    );
  }

  return (
    <>
      {showOwnerRepo && (
        <div className="mb-5">
          <div className={SECTION_LABEL_CLASS}>Repository</div>
          <div className="grid grid-cols-2 gap-2.5">
            <div>
              <Label className={FIELD_LABEL_CLASS}>Owner</Label>
              <Input
                type="text"
                value={owner ?? ''}
                onChange={(e) => setOwner?.(e.target.value)}
                placeholder="e.g. acme-org"
                className="text-xs tracking-[0.03em] rounded-none h-8"
              />
            </div>
            <div>
              <Label className={FIELD_LABEL_CLASS}>Repo</Label>
              <Input
                type="text"
                value={repo ?? ''}
                onChange={(e) => setRepo?.(e.target.value)}
                placeholder="e.g. my-project"
                className="text-xs tracking-[0.03em] rounded-none h-8"
              />
            </div>
          </div>
        </div>
      )}

      <div className="mb-5">
        <div className={SECTION_LABEL_CLASS}>Labels</div>
        <div className="flex flex-col gap-2.5">
          {textField('Approved', 'label_approved')}
          {textField('In Progress', 'label_in_progress')}
          {textField('Done', 'label_done')}
          {textField('Failed', 'label_failed')}
          {textField('Feature Request', 'label_feature_request')}
        </div>
      </div>

      <div className="mb-5">
        <div className={SECTION_LABEL_CLASS}>Limits</div>
        <div className="grid grid-cols-3 gap-2.5">
          {numField('Vote threshold', 'vote_threshold')}
          {numField('Timeout (min)', 'timeout_minutes')}
          {numField('Budget (USD)', 'max_budget_usd')}
        </div>
      </div>

      <div>
        <div className={SECTION_LABEL_CLASS}>Anthropic</div>
        <div>
          <Label className={FIELD_LABEL_CLASS}>
            API Key <span className="text-muted-foreground tracking-[0.08em]">(leave blank to keep)</span>
          </Label>
          <Input
            type="password"
            placeholder="sk-ant-…"
            onChange={(e) =>
              setForm((f) => ({
                ...f,
                anthropic_api_key: e.target.value || undefined,
              }))
            }
            className="text-xs tracking-[0.03em] rounded-none h-8"
          />
        </div>
      </div>

      <div>
        <div className={SECTION_LABEL_CLASS}>Community Board</div>
        <div className="flex items-center justify-between py-2.5 px-3 bg-muted/50 border border-border rounded-lg">
          <div>
            <span className="text-[11px] text-foreground tracking-[0.03em] block">
              Public board
            </span>
            <span className="text-[10px] text-muted-foreground tracking-[0.03em]">
              Allow community to propose and vote on features
            </span>
          </div>
          <Switch
            checked={form.is_board_public}
            onCheckedChange={(checked) =>
              setForm((f) => ({ ...f, is_board_public: checked }))
            }
          />
        </div>
      </div>
    </>
  );
}

// ─── Modal footer ─────────────────────────────────────────────────────────────

function ModalFooter({
  onCancel,
  onSubmit,
  isPending,
  submitLabel,
  error,
  destructive,
}: {
  onCancel: () => void;
  onSubmit: () => void;
  isPending: boolean;
  submitLabel: string;
  error?: string;
  destructive?: boolean;
}) {
  return (
    <>
      {error && (
        <p className="mb-3 text-[11px] text-destructive tracking-[0.04em]">{error}</p>
      )}
      <DialogFooter className="border-border pt-4">
        <Button variant="ghost" onClick={onCancel} className="text-[10px] tracking-[0.12em] uppercase">
          Cancel
        </Button>
        <Button
          onClick={onSubmit}
          disabled={isPending}
          variant={destructive ? 'destructive' : 'default'}
          className="text-[10px] tracking-[0.15em] uppercase"
        >
          {isPending ? '…' : submitLabel}
        </Button>
      </DialogFooter>
    </>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

type ModalState =
  | { kind: 'none' }
  | { kind: 'create' }
  | { kind: 'edit'; config: RepoConfig }
  | { kind: 'delete'; config: RepoConfig };

export default function ConfigPage() {
  const [modal, setModal] = useState<ModalState>({ kind: 'none' });

  const { data: repos, isLoading, error } = useQuery({
    queryKey: ['repos'],
    queryFn: () => listRepos(),
  });

  if (isLoading) {
    return <p className="text-xs text-muted-foreground tracking-[0.08em]">Loading…</p>;
  }

  if (error) {
    return (
      <p className="text-xs text-destructive tracking-[0.04em]">
        error: {error instanceof Error ? error.message : 'unknown'}
      </p>
    );
  }

  return (
    <div className="animate-slide-up">
      {/* Header */}
      <div className="flex items-baseline justify-between mb-6">
        <div className="flex items-baseline gap-3">
          <span className="text-[10px] tracking-[0.25em] uppercase text-primary">Config</span>
          {repos && (
            <span className="text-[10px] text-muted-foreground tracking-[0.05em]">
              {repos.length} repos
            </span>
          )}
        </div>
        <Button
          variant="outline"
          onClick={() => setModal({ kind: 'create' })}
          className="text-[10px] tracking-[0.15em] uppercase"
        >
          + Add
        </Button>
      </div>

      {/* Table */}
      {repos?.length === 0 ? (
        <div className="py-8 border-t border-b border-border">
          <p className="text-xs text-muted-foreground tracking-[0.06em] text-center">
            no repos configured
          </p>
          <p className="mt-2 text-[11px] text-muted-foreground tracking-[0.06em] text-center">
            add one to override service defaults for a specific repository
          </p>
        </div>
      ) : (
        <div>
          {/* Column headers */}
          <div className="grid gap-x-5 pb-2 border-b border-border grid-cols-[2fr_80px_80px_80px_100px_72px]">
            {['Repository', 'Threshold', 'Timeout', 'Budget', 'Updated', ''].map((h) => (
              <span
                key={h}
                className="text-[10px] tracking-[0.15em] uppercase text-muted-foreground"
              >
                {h}
              </span>
            ))}
          </div>

          {repos?.map((cfg: RepoConfig) => (
            <div
              key={cfg.id}
              className="grid gap-x-5 items-center py-2.5 border-b border-border hover:bg-muted/30 transition-colors grid-cols-[2fr_80px_80px_80px_100px_72px]"
            >
              <span className="text-xs text-foreground tracking-[0.02em]">
                {cfg.owner}/{cfg.repo}
              </span>
              <span className="text-xs text-muted-foreground tracking-[0.04em]">
                {cfg.vote_threshold}
              </span>
              <span className="text-xs text-muted-foreground tracking-[0.04em]">
                {cfg.timeout_minutes}m
              </span>
              <span className="text-xs text-muted-foreground tracking-[0.04em]">
                ${cfg.max_budget_usd}
              </span>
              <span className="text-[11px] text-muted-foreground tracking-[0.04em] whitespace-nowrap">
                {new Date(cfg.updated_at).toLocaleDateString()}
              </span>

              <div className="flex items-center justify-end gap-3">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setModal({ kind: 'edit', config: cfg })}
                  className="text-[10px] tracking-[0.12em] uppercase"
                >
                  Edit
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setModal({ kind: 'delete', config: cfg })}
                  className="text-[10px] tracking-[0.12em] uppercase text-destructive hover:text-destructive"
                >
                  Del
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Modals */}
      <Dialog
        open={modal.kind !== 'none'}
        onOpenChange={(open) => !open && setModal({ kind: 'none' })}
      >
        <DialogContent className="sm:max-w-[440px] max-h-[90vh] overflow-y-auto">
          {modal.kind === 'create' && (
            <ConfigCreateModal
              onClose={() => setModal({ kind: 'none' })}
            />
          )}
          {modal.kind === 'edit' && modal.config && (
            <ConfigEditModal
              config={modal.config}
              onClose={() => setModal({ kind: 'none' })}
            />
          )}
          {modal.kind === 'delete' && modal.config && (
            <ConfigDeleteModal
              config={modal.config}
              onClose={() => setModal({ kind: 'none' })}
            />
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ─── Create modal content ─────────────────────────────────────────────────────

function ConfigCreateModal({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient();
  const [owner, setOwner] = useState('');
  const [repo, setRepo] = useState('');
  const [form, setForm] = useState<UpdateRepoConfigRequest>({ ...DEFAULT_FORM });

  const create = useMutation({
    mutationFn: () =>
      updateRepoConfig(owner.trim(), repo.trim(), form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['repos'] });
      onClose();
    },
  });

  const errorMsg = create.error instanceof Error ? create.error.message : undefined;
  const canSubmit = owner.trim() !== '' && repo.trim() !== '';

  return (
    <>
      <DialogHeader>
        <DialogTitle className="text-[10px] tracking-[0.2em] uppercase text-muted-foreground font-normal">
          Add Config
        </DialogTitle>
      </DialogHeader>
      <FormFields
        form={form}
        setForm={setForm}
        showOwnerRepo
        owner={owner}
        repo={repo}
        setOwner={setOwner}
        setRepo={setRepo}
      />
      <ModalFooter
        onCancel={onClose}
        onSubmit={() => {
          if (canSubmit) create.mutate();
        }}
        isPending={create.isPending}
        submitLabel="Add"
        error={errorMsg}
      />
    </>
  );
}

// ─── Edit modal content ───────────────────────────────────────────────────────

function ConfigEditModal({ config, onClose }: { config: RepoConfig; onClose: () => void }) {
  const qc = useQueryClient();
  const [form, setForm] = useState<UpdateRepoConfigRequest>({
    label_approved: config.label_approved,
    label_in_progress: config.label_in_progress,
    label_done: config.label_done,
    label_failed: config.label_failed,
    label_feature_request: config.label_feature_request,
    vote_threshold: config.vote_threshold,
    timeout_minutes: config.timeout_minutes,
    max_budget_usd: config.max_budget_usd,
    is_board_public: config.is_board_public,
  });

  const update = useMutation({
    mutationFn: () =>
      updateRepoConfig(config.owner, config.repo, form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['repos'] });
      onClose();
    },
  });

  const errorMsg = update.error instanceof Error ? update.error.message : undefined;

  return (
    <>
      <DialogHeader>
        <DialogTitle className="text-[10px] tracking-[0.2em] uppercase text-muted-foreground font-normal">
          Edit Config
        </DialogTitle>
        <span className="text-[13px] text-foreground tracking-[0.02em]">
          {config.owner}/{config.repo}
        </span>
      </DialogHeader>
      <FormFields form={form} setForm={setForm} />
      <ModalFooter
        onCancel={onClose}
        onSubmit={() => update.mutate()}
        isPending={update.isPending}
        submitLabel="Save"
        error={errorMsg}
      />
    </>
  );
}

// ─── Delete confirm modal content ──────────────────────────────────────────────

function ConfigDeleteModal({ config, onClose }: { config: RepoConfig; onClose: () => void }) {
  const qc = useQueryClient();

  const del = useMutation({
    mutationFn: () =>
      deleteRepoConfig(config.owner, config.repo),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['repos'] });
      onClose();
    },
  });

  const errorMsg = del.error instanceof Error ? del.error.message : undefined;

  return (
    <>
      <DialogHeader>
        <DialogTitle className="text-[10px] tracking-[0.2em] uppercase text-muted-foreground font-normal">
          Delete Config
        </DialogTitle>
        <span className="text-[13px] text-foreground tracking-[0.02em]">
          {config.owner}/{config.repo}
        </span>
      </DialogHeader>
      <p className="text-xs text-muted-foreground tracking-[0.03em] leading-relaxed mb-5">
        This will permanently remove the configuration for{' '}
        <span className="text-foreground">{config.owner}/{config.repo}</span>
        . The repository's webhook trigger will fall back to service defaults.
      </p>
      <ModalFooter
        onCancel={onClose}
        onSubmit={() => del.mutate()}
        isPending={del.isPending}
        submitLabel="Delete"
        error={errorMsg}
        destructive
      />
    </>
  );
}
