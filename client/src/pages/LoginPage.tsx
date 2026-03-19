import { useState, type FormEvent } from 'react';
import { useAuth } from '../lib/auth';

export default function LoginPage() {
  const { login } = useAuth();
  const [key, setKey] = useState('');
  const [error, setError] = useState('');

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = key.trim();
    if (!trimmed) {
      setError('api key required');
      return;
    }
    login(trimmed);
  }

  return (
    <div className="min-h-screen bg-gray-950 flex items-center justify-center">
      {/* Faint corner decoration */}
      <div className="fixed top-6 left-6 text-[10px] tracking-[0.3em] text-border uppercase">
        github-vote-llm
      </div>
      <div className="fixed bottom-6 right-6 text-[10px] tracking-[0.2em] text-border uppercase">
        v1.0
      </div>

      <div className="animate-slide-up w-[300px]">
        {/* Wordmark */}
        <div className="mb-10 text-center">
          <div className="text-[11px] tracking-[0.45em] uppercase text-emerald-400 font-semibold mb-2">
            vote-llm
          </div>
          <div className="w-6 h-px bg-gray-800 mx-auto" />
          <div className="mt-2 text-[10px] tracking-[0.2em] text-gray-500 uppercase">
            authentication required
          </div>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="flex flex-col gap-2">
          <div>
            <input
              type="password"
              value={key}
              onChange={(e) => {
                setKey(e.target.value);
                setError('');
              }}
              placeholder="X-Api-Key"
              autoFocus
              className="w-full py-2.5 px-3 bg-gray-900 border border-gray-800 text-gray-100 text-xs tracking-[0.04em] outline-none rounded-none box-border transition-[border-color] duration-150 focus:border-gray-500"
            />
            {error && (
              <p className="mt-1.5 text-[11px] text-red tracking-[0.04em]">{error}</p>
            )}
          </div>
          <button
            type="submit"
            className="w-full py-2.5 px-3 bg-emerald-400 text-gray-950 text-[11px] tracking-[0.2em] uppercase font-semibold border-none cursor-pointer rounded-none transition-opacity duration-150 hover:opacity-[0.85]"
          >
            Continue
          </button>
        </form>
      </div>
    </div>
  );
}
