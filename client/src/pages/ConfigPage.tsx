import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listRepos, updateRepoConfig, deleteRepoConfig } from '../client/sdk.gen';
import type { RepoConfig, UpdateRepoConfigRequest } from '../client/types.gen';

// ─── Shared styles ────────────────────────────────────────────────────────────

const INPUT_STYLE: React.CSSProperties = {
  width: '100%',
  padding: '8px 10px',
  background: '#0C0C0C',
  border: '1px solid #191919',
  color: '#C4C0AC',
  fontSize: 12,
  letterSpacing: '0.03em',
  outline: 'none',
  borderRadius: 0,
  boxSizing: 'border-box',
  transition: 'border-color 150ms',
};

const SECTION_LABEL_STYLE: React.CSSProperties = {
  fontSize: 9,
  letterSpacing: '0.25em',
  textTransform: 'uppercase',
  color: '#201E1A',
  marginBottom: 12,
};

const FIELD_LABEL_STYLE: React.CSSProperties = {
  display: 'block',
  fontSize: 10,
  letterSpacing: '0.15em',
  textTransform: 'uppercase',
  color: '#302E2A',
  marginBottom: 5,
};

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
        <label style={FIELD_LABEL_STYLE}>{label}</label>
        <input
          type="text"
          value={String(form[key] ?? '')}
          onChange={(e) => setForm((f) => ({ ...f, [key]: e.target.value }))}
          style={INPUT_STYLE}
          onFocus={(e) => (e.target.style.borderColor = '#302E2A')}
          onBlur={(e) => (e.target.style.borderColor = '#191919')}
        />
      </div>
    );
  }

  function numField(label: string, key: keyof UpdateRepoConfigRequest) {
    return (
      <div key={key}>
        <label style={FIELD_LABEL_STYLE}>{label}</label>
        <input
          type="number"
          value={String(form[key] ?? '')}
          onChange={(e) => setForm((f) => ({ ...f, [key]: Number(e.target.value) }))}
          style={INPUT_STYLE}
          onFocus={(e) => (e.target.style.borderColor = '#302E2A')}
          onBlur={(e) => (e.target.style.borderColor = '#191919')}
        />
      </div>
    );
  }

  return (
    <>
      {showOwnerRepo && (
        <div style={{ marginBottom: 20 }}>
          <div style={SECTION_LABEL_STYLE}>Repository</div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
            <div>
              <label style={FIELD_LABEL_STYLE}>Owner</label>
              <input
                type="text"
                value={owner ?? ''}
                onChange={(e) => setOwner?.(e.target.value)}
                placeholder="e.g. acme-org"
                style={{ ...INPUT_STYLE, color: owner ? '#C4C0AC' : undefined }}
                onFocus={(e) => (e.target.style.borderColor = '#302E2A')}
                onBlur={(e) => (e.target.style.borderColor = '#191919')}
              />
            </div>
            <div>
              <label style={FIELD_LABEL_STYLE}>Repo</label>
              <input
                type="text"
                value={repo ?? ''}
                onChange={(e) => setRepo?.(e.target.value)}
                placeholder="e.g. my-project"
                style={{ ...INPUT_STYLE, color: repo ? '#C4C0AC' : undefined }}
                onFocus={(e) => (e.target.style.borderColor = '#302E2A')}
                onBlur={(e) => (e.target.style.borderColor = '#191919')}
              />
            </div>
          </div>
        </div>
      )}

      <div style={{ marginBottom: 20 }}>
        <div style={SECTION_LABEL_STYLE}>Labels</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {textField('Approved', 'label_approved')}
          {textField('In Progress', 'label_in_progress')}
          {textField('Done', 'label_done')}
          {textField('Failed', 'label_failed')}
          {textField('Feature Request', 'label_feature_request')}
        </div>
      </div>

      <div style={{ marginBottom: 20 }}>
        <div style={SECTION_LABEL_STYLE}>Limits</div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10 }}>
          {numField('Vote threshold', 'vote_threshold')}
          {numField('Timeout (min)', 'timeout_minutes')}
          {numField('Budget (USD)', 'max_budget_usd')}
        </div>
      </div>

      <div>
        <div style={SECTION_LABEL_STYLE}>Anthropic</div>
        <div>
          <label style={FIELD_LABEL_STYLE}>
            API Key{' '}
            <span style={{ color: '#201E1A', letterSpacing: '0.08em' }}>
              (leave blank to keep)
            </span>
          </label>
          <input
            type="password"
            placeholder="sk-ant-…"
            onChange={(e) =>
              setForm((f) => ({
                ...f,
                anthropic_api_key: e.target.value || undefined,
              }))
            }
            style={{ ...INPUT_STYLE, color: '#403C34' }}
            onFocus={(e) => (e.target.style.borderColor = '#302E2A')}
            onBlur={(e) => (e.target.style.borderColor = '#191919')}
          />
        </div>
      </div>

      <div>
        <div style={SECTION_LABEL_STYLE}>Community Board</div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '10px 12px',
            background: '#080808',
            border: '1px solid #141414',
          }}
        >
          <div>
            <span style={{ fontSize: 11, color: '#6A6458', letterSpacing: '0.03em', display: 'block' }}>
              Public board
            </span>
            <span style={{ fontSize: 10, color: '#302E2A', letterSpacing: '0.03em' }}>
              Allow community to propose and vote on features
            </span>
          </div>
          <button
            onClick={() => setForm((f) => ({ ...f, is_board_public: !f.is_board_public }))}
            style={{
              width: 36,
              height: 20,
              borderRadius: 10,
              background: form.is_board_public ? '#00E87A' : '#191919',
              border: 'none',
              cursor: 'pointer',
              position: 'relative',
              flexShrink: 0,
              transition: 'background 200ms',
            }}
          >
            <span
              style={{
                position: 'absolute',
                top: 2,
                left: form.is_board_public ? 18 : 2,
                width: 16,
                height: 16,
                borderRadius: '50%',
                background: form.is_board_public ? '#070707' : '#302E2A',
                transition: 'left 200ms, background 200ms',
              }}
            />
          </button>
        </div>
      </div>
    </>
  );
}

// ─── Modal shell ──────────────────────────────────────────────────────────────

function Modal({
  title,
  subtitle,
  onClose,
  children,
}: {
  title: string;
  subtitle?: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.8)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 50,
      }}
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        className="animate-slide-up"
        style={{
          background: '#0C0C0C',
          border: '1px solid #191919',
          padding: 24,
          width: '100%',
          maxWidth: 440,
          maxHeight: '90vh',
          overflowY: 'auto',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'baseline',
            justifyContent: 'space-between',
            marginBottom: 20,
            paddingBottom: 16,
            borderBottom: '1px solid #191919',
          }}
        >
          <div>
            <span
              style={{
                display: 'block',
                fontSize: 10,
                letterSpacing: '0.2em',
                textTransform: 'uppercase',
                color: '#302E2A',
                marginBottom: subtitle ? 3 : 0,
              }}
            >
              {title}
            </span>
            {subtitle && (
              <span style={{ fontSize: 13, color: '#C4C0AC', letterSpacing: '0.02em' }}>
                {subtitle}
              </span>
            )}
          </div>
          <button
            onClick={onClose}
            style={{
              fontSize: 18,
              color: '#302E2A',
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              lineHeight: 1,
              padding: '0 0 0 16px',
              transition: 'color 150ms',
            }}
            onMouseEnter={(e) => ((e.target as HTMLElement).style.color = '#6A6458')}
            onMouseLeave={(e) => ((e.target as HTMLElement).style.color = '#302E2A')}
          >
            ×
          </button>
        </div>
        {children}
      </div>
    </div>
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
        <p
          style={{
            marginBottom: 12,
            fontSize: 11,
            color: '#FF3A3A',
            letterSpacing: '0.04em',
          }}
        >
          {error}
        </p>
      )}
      <div
        style={{
          display: 'flex',
          justifyContent: 'flex-end',
          gap: 10,
          paddingTop: 16,
          borderTop: '1px solid #191919',
        }}
      >
        <button
          onClick={onCancel}
          style={{
            padding: '8px 14px',
            background: 'none',
            border: 'none',
            fontSize: 10,
            letterSpacing: '0.12em',
            textTransform: 'uppercase',
            color: '#302E2A',
            cursor: 'pointer',
            transition: 'color 150ms',
          }}
          onMouseEnter={(e) => ((e.target as HTMLElement).style.color = '#6A6458')}
          onMouseLeave={(e) => ((e.target as HTMLElement).style.color = '#302E2A')}
        >
          Cancel
        </button>
        <button
          onClick={onSubmit}
          disabled={isPending}
          style={{
            padding: '8px 16px',
            background: destructive ? '#3A1414' : '#00E87A',
            color: destructive ? '#FF3A3A' : '#070707',
            fontSize: 10,
            letterSpacing: '0.15em',
            textTransform: 'uppercase',
            fontWeight: 600,
            border: destructive ? '1px solid #FF3A3A40' : 'none',
            cursor: isPending ? 'not-allowed' : 'pointer',
            opacity: isPending ? 0.5 : 1,
            transition: 'opacity 150ms',
          }}
          onMouseEnter={(e) => {
            if (!isPending) (e.target as HTMLElement).style.opacity = '0.85';
          }}
          onMouseLeave={(e) => {
            if (!isPending) (e.target as HTMLElement).style.opacity = '1';
          }}
        >
          {isPending ? '…' : submitLabel}
        </button>
      </div>
    </>
  );
}

// ─── Create modal ─────────────────────────────────────────────────────────────

function CreateModal({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient();
  const [owner, setOwner] = useState('');
  const [repo, setRepo] = useState('');
  const [form, setForm] = useState<UpdateRepoConfigRequest>({ ...DEFAULT_FORM });

  const create = useMutation({
    mutationFn: () =>
      updateRepoConfig({ path: { owner: owner.trim(), repo: repo.trim() }, body: form }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['repos'] });
      onClose();
    },
  });

  const errorMsg = create.error instanceof Error ? create.error.message : undefined;
  const canSubmit = owner.trim() !== '' && repo.trim() !== '';

  return (
    <Modal title="Add Config" onClose={onClose}>
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
    </Modal>
  );
}

// ─── Edit modal ───────────────────────────────────────────────────────────────

function EditModal({ config, onClose }: { config: RepoConfig; onClose: () => void }) {
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
      updateRepoConfig({ path: { owner: config.owner, repo: config.repo }, body: form }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['repos'] });
      onClose();
    },
  });

  const errorMsg = update.error instanceof Error ? update.error.message : undefined;

  return (
    <Modal title="Edit Config" subtitle={`${config.owner}/${config.repo}`} onClose={onClose}>
      <FormFields form={form} setForm={setForm} />
      <ModalFooter
        onCancel={onClose}
        onSubmit={() => update.mutate()}
        isPending={update.isPending}
        submitLabel="Save"
        error={errorMsg}
      />
    </Modal>
  );
}

// ─── Delete confirm modal ─────────────────────────────────────────────────────

function DeleteModal({ config, onClose }: { config: RepoConfig; onClose: () => void }) {
  const qc = useQueryClient();

  const del = useMutation({
    mutationFn: () =>
      deleteRepoConfig({ path: { owner: config.owner, repo: config.repo } }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['repos'] });
      onClose();
    },
  });

  const errorMsg = del.error instanceof Error ? del.error.message : undefined;

  return (
    <Modal title="Delete Config" subtitle={`${config.owner}/${config.repo}`} onClose={onClose}>
      <p
        style={{
          fontSize: 12,
          color: '#6A6458',
          letterSpacing: '0.03em',
          lineHeight: 1.7,
          marginBottom: 20,
        }}
      >
        This will permanently remove the configuration for{' '}
        <span style={{ color: '#C4C0AC' }}>
          {config.owner}/{config.repo}
        </span>
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
    </Modal>
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
    queryFn: () => listRepos().then((r) => r.data ?? []),
  });

  if (isLoading) {
    return (
      <p style={{ fontSize: 12, color: '#302E2A', letterSpacing: '0.08em' }}>Loading…</p>
    );
  }

  if (error) {
    return (
      <p style={{ fontSize: 12, color: '#FF3A3A', letterSpacing: '0.04em' }}>
        error: {error instanceof Error ? error.message : 'unknown'}
      </p>
    );
  }

  return (
    <div className="animate-slide-up">
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'baseline',
          justifyContent: 'space-between',
          marginBottom: 24,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 12 }}>
          <span
            style={{
              fontSize: 10,
              letterSpacing: '0.25em',
              textTransform: 'uppercase',
              color: '#00E87A',
            }}
          >
            Config
          </span>
          {repos && (
            <span style={{ fontSize: 10, color: '#302E2A', letterSpacing: '0.05em' }}>
              {repos.length} repos
            </span>
          )}
        </div>
        <button
          onClick={() => setModal({ kind: 'create' })}
          style={{
            padding: '6px 12px',
            background: 'none',
            border: '1px solid #191919',
            fontSize: 10,
            letterSpacing: '0.15em',
            textTransform: 'uppercase',
            color: '#403C34',
            cursor: 'pointer',
            transition: 'border-color 150ms, color 150ms',
          }}
          onMouseEnter={(e) => {
            const el = e.currentTarget;
            el.style.borderColor = '#302E2A';
            el.style.color = '#C4C0AC';
          }}
          onMouseLeave={(e) => {
            const el = e.currentTarget;
            el.style.borderColor = '#191919';
            el.style.color = '#403C34';
          }}
        >
          + Add
        </button>
      </div>

      {/* Table */}
      {repos?.length === 0 ? (
        <div
          style={{
            padding: '32px 0',
            borderTop: '1px solid #191919',
            borderBottom: '1px solid #191919',
          }}
        >
          <p
            style={{
              fontSize: 12,
              color: '#302E2A',
              letterSpacing: '0.06em',
              textAlign: 'center',
            }}
          >
            no repos configured
          </p>
          <p
            style={{
              marginTop: 8,
              fontSize: 11,
              color: '#201E1A',
              letterSpacing: '0.06em',
              textAlign: 'center',
            }}
          >
            add one to override service defaults for a specific repository
          </p>
        </div>
      ) : (
        <div>
          {/* Column headers */}
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: '2fr 80px 80px 80px 100px 72px',
              gap: '0 20px',
              paddingBottom: 8,
              borderBottom: '1px solid #191919',
            }}
          >
            {['Repository', 'Threshold', 'Timeout', 'Budget', 'Updated', ''].map((h) => (
              <span
                key={h}
                style={{
                  fontSize: 10,
                  letterSpacing: '0.15em',
                  textTransform: 'uppercase',
                  color: '#302E2A',
                }}
              >
                {h}
              </span>
            ))}
          </div>

          {repos?.map((cfg: RepoConfig) => (
            <div
              key={cfg.id}
              style={{
                display: 'grid',
                gridTemplateColumns: '2fr 80px 80px 80px 100px 72px',
                gap: '0 20px',
                alignItems: 'center',
                padding: '11px 0',
                borderBottom: '1px solid #111111',
                transition: 'background 150ms',
              }}
              onMouseEnter={(e) =>
                ((e.currentTarget as HTMLElement).style.background = '#0C0C0C')
              }
              onMouseLeave={(e) =>
                ((e.currentTarget as HTMLElement).style.background = 'transparent')
              }
            >
              <span
                style={{ fontSize: 12, color: '#C4C0AC', letterSpacing: '0.02em' }}
              >
                {cfg.owner}/{cfg.repo}
              </span>
              <span style={{ fontSize: 12, color: '#403C34', letterSpacing: '0.04em' }}>
                {cfg.vote_threshold}
              </span>
              <span style={{ fontSize: 12, color: '#403C34', letterSpacing: '0.04em' }}>
                {cfg.timeout_minutes}m
              </span>
              <span style={{ fontSize: 12, color: '#403C34', letterSpacing: '0.04em' }}>
                ${cfg.max_budget_usd}
              </span>
              <span
                style={{
                  fontSize: 11,
                  color: '#302E2A',
                  letterSpacing: '0.04em',
                  whiteSpace: 'nowrap',
                }}
              >
                {new Date(cfg.updated_at).toLocaleDateString()}
              </span>

              {/* Row actions */}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'flex-end',
                  gap: 12,
                }}
              >
                <button
                  onClick={() => setModal({ kind: 'edit', config: cfg })}
                  style={{
                    fontSize: 10,
                    letterSpacing: '0.12em',
                    textTransform: 'uppercase',
                    color: '#302E2A',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    transition: 'color 150ms',
                  }}
                  onMouseEnter={(e) => ((e.target as HTMLElement).style.color = '#C4C0AC')}
                  onMouseLeave={(e) => ((e.target as HTMLElement).style.color = '#302E2A')}
                >
                  Edit
                </button>
                <button
                  onClick={() => setModal({ kind: 'delete', config: cfg })}
                  style={{
                    fontSize: 10,
                    letterSpacing: '0.12em',
                    textTransform: 'uppercase',
                    color: '#2A1414',
                    background: 'none',
                    border: 'none',
                    cursor: 'pointer',
                    transition: 'color 150ms',
                  }}
                  onMouseEnter={(e) => ((e.target as HTMLElement).style.color = '#FF3A3A')}
                  onMouseLeave={(e) => ((e.target as HTMLElement).style.color = '#2A1414')}
                >
                  Del
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Modals */}
      {modal.kind === 'create' && (
        <CreateModal onClose={() => setModal({ kind: 'none' })} />
      )}
      {modal.kind === 'edit' && (
        <EditModal config={modal.config} onClose={() => setModal({ kind: 'none' })} />
      )}
      {modal.kind === 'delete' && (
        <DeleteModal config={modal.config} onClose={() => setModal({ kind: 'none' })} />
      )}
    </div>
  );
}
