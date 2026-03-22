import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { createOrganization } from '../lib/api';
import { ApiError } from '../lib/api';

interface CreateOrganizationPageProps {
  onCreated?: () => void;
}

export default function CreateOrganizationPage({ onCreated }: CreateOrganizationPageProps) {
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;

    setError(null);
    setIsSubmitting(true);
    try {
      await createOrganization(trimmed);
      onCreated?.();
      navigate('/', { replace: true });
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 400 && err.body && typeof err.body === 'object' && 'error' in err.body) {
          const msg = String((err.body as { error: string }).error);
          if (msg.includes('already belong')) {
            onCreated?.();
            navigate('/', { replace: true });
            return;
          }
          setError(msg);
        } else {
          setError(err.message);
        }
      } else {
        setError('Failed to create organization');
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className="animate-slide-up w-[320px]">
        <div className="mb-10 text-center">
          <h1 className="text-[24px] font-bold text-foreground mb-2">
            Create your organization
          </h1>
          <p className="text-[15px] text-muted-foreground">
            Give your organization a name to get started.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <div>
            <input
              type="text"
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                setError(null);
              }}
              placeholder="Organization name"
              autoComplete="organization"
              className="w-full py-3 px-4 bg-background border border-input text-foreground text-[15px] rounded-[8px] outline-none box-border transition-colors duration-150 focus:border-ring focus:ring-2 focus:ring-ring/20"
            />
          </div>
          {error && (
            <p className="text-[14px] text-destructive">{error}</p>
          )}
          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full py-3 px-4 bg-primary text-primary-foreground text-[15px] font-semibold rounded-[8px] border-none cursor-pointer transition-opacity duration-150 hover:opacity-90 disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {isSubmitting ? '…' : 'Create organization'}
          </button>
        </form>
      </div>
    </div>
  );
}
