import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { createOrganization, ApiError } from '../lib/api';
import { slugify } from '../lib/utils';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import { Alert, AlertDescription } from '@/components/ui/alert';

interface CreateOrganizationPageProps {
  onCreated?: () => void;
}

export default function CreateOrganizationPage({ onCreated }: CreateOrganizationPageProps) {
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [slugManuallyEdited, setSlugManuallyEdited] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  function handleNameChange(value: string) {
    setName(value);
    if (!slugManuallyEdited) {
      setSlug(slugify(value));
    }
    setError(null);
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;

    setError(null);
    setIsSubmitting(true);
    try {
      await createOrganization(trimmed, slug.trim() || undefined);
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
      <div className="animate-slide-up w-full max-w-sm">
        <Card className="px-4 py-4 sm:px-6 sm:py-6">
          <CardHeader className="gap-2">
            <CardTitle>Create your organization</CardTitle>
            <CardDescription>Give your organization a name to get started.</CardDescription>
          </CardHeader>
          <CardContent className="pt-6">
            <form onSubmit={handleSubmit}>
              <FieldGroup className="gap-4">
                <Field>
                  <FieldLabel htmlFor="org-name">Organization name</FieldLabel>
                  <Input
                    id="org-name"
                    type="text"
                    value={name}
                    onChange={(e) => handleNameChange(e.target.value)}
                    placeholder="Organization name"
                    autoComplete="organization"
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="org-slug">
                    URL slug
                    <span className="ml-1.5 text-xs text-muted-foreground font-normal">
                      Used in your portal URL
                    </span>
                  </FieldLabel>
                  <Input
                    id="org-slug"
                    type="text"
                    value={slug}
                    onChange={(e) => {
                      setSlug(e.target.value);
                      setSlugManuallyEdited(true);
                      setError(null);
                    }}
                    placeholder="my-organization"
                  />
                </Field>
                {error && (
                  <Alert variant="danger">
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}
                <Button type="submit" disabled={isSubmitting} className="w-full">
                  {isSubmitting ? '…' : 'Create organization'}
                </Button>
              </FieldGroup>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
