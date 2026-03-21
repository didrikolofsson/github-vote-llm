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
    <div className="min-h-screen bg-gray-950 flex items-center justify-center">
      <div className="fixed top-6 left-6 text-[10px] tracking-[0.3em] text-border uppercase">
        github-vote-llm
      </div>
      <div className="fixed bottom-6 right-6 text-[10px] tracking-[0.2em] text-border uppercase">
        v1.0
      </div>

      <div className="animate-slide-up w-[300px]">
        <div className="mb-10 text-center">
          <div className="text-[11px] tracking-[0.45em] uppercase text-emerald-400 font-semibold mb-2">
            vote-llm
          </div>
          <div className="w-6 h-px bg-gray-800 mx-auto" />
          <div className="mt-2 text-[10px] tracking-[0.2em] text-gray-500 uppercase">
            {mode === 'signup' ? 'Create account' : 'Sign in'}
          </div>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-2">
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
              className="w-full py-2.5 px-3 bg-gray-900 border border-gray-800 text-gray-100 text-xs tracking-[0.04em] outline-none rounded-none box-border transition-[border-color] duration-150 focus:border-gray-500"
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
              className="w-full py-2.5 px-3 bg-gray-900 border border-gray-800 text-gray-100 text-xs tracking-[0.04em] outline-none rounded-none box-border transition-[border-color] duration-150 focus:border-gray-500"
            />
          </div>
          {error && (
            <p className="text-[11px] text-red-500 tracking-[0.04em]">{error}</p>
          )}
          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full py-2.5 px-3 bg-emerald-400 text-gray-950 text-[11px] tracking-[0.2em] uppercase font-semibold border-none cursor-pointer rounded-none transition-opacity duration-150 hover:opacity-[0.85] disabled:opacity-60 disabled:cursor-not-allowed"
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
          className="mt-4 w-full text-[10px] tracking-[0.12em] uppercase text-gray-500 bg-transparent border-none cursor-pointer hover:text-gray-400 transition-colors"
        >
          {mode === 'login' ? 'Create an account' : 'Already have an account? Sign in'}
        </button>
      </div>
    </div>
  );
}
