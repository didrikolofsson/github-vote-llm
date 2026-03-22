import { useState, type FormEvent } from 'react';
import { useAuth } from '../lib/auth';

export default function LoginPage() {
  const { login, signup, error, clearError } = useAuth();
  const [mode, setMode] = useState<'login' | 'signup'>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmedEmail = email.trim();
    const trimmedPassword = password.trim();
    if (!trimmedEmail || !trimmedPassword) return;

    clearError();
    setIsSubmitting(true);
    try {
      if (mode === 'signup') {
        await signup(trimmedEmail, trimmedPassword);
      } else {
        await login(trimmedEmail, trimmedPassword);
      }
    } catch {
      // Error is set in auth context
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className="fixed top-6 left-6 text-xs text-muted-foreground">
        github-vote-llm
      </div>
      <div className="fixed bottom-6 right-6 text-xs text-muted-foreground">
        v1.0
      </div>

      <div className="animate-slide-up w-[320px]">
        <div className="mb-10 text-center">
          <div className="text-primary font-bold text-[24px] mb-2">
            vote-llm
          </div>
          <div className="w-8 h-px bg-border mx-auto" />
          <div className="mt-3 text-[15px] text-muted-foreground">
            {mode === 'signup' ? 'Create account' : 'Sign in'}
          </div>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <div>
            <input
              type="email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                clearError();
              }}
              placeholder="Email"
              autoComplete="email"
              className="w-full py-3 px-4 bg-background border border-input text-foreground text-[15px] rounded-[8px] outline-none box-border transition-colors duration-150 focus:border-ring focus:ring-2 focus:ring-ring/20"
            />
          </div>
          <div>
            <input
              type="password"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value);
                clearError();
              }}
              placeholder="Password"
              autoComplete={mode === 'signup' ? 'new-password' : 'current-password'}
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
            {isSubmitting ? '…' : mode === 'signup' ? 'Sign up' : 'Continue'}
          </button>
        </form>

        <button
          type="button"
          onClick={() => {
            setMode((m) => (m === 'login' ? 'signup' : 'login'));
            clearError();
          }}
          className="mt-4 w-full text-[15px] text-muted-foreground bg-transparent border-none cursor-pointer hover:text-foreground transition-colors"
        >
          {mode === 'login' ? 'Create an account' : 'Already have an account? Sign in'}
        </button>
      </div>
    </div>
  );
}
