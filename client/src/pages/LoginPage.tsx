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
    <div
      style={{
        minHeight: '100vh',
        background: '#070707',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      {/* Faint corner decoration */}
      <div
        style={{
          position: 'fixed',
          top: 24,
          left: 24,
          fontSize: 10,
          letterSpacing: '0.3em',
          color: '#191919',
          textTransform: 'uppercase',
        }}
      >
        github-vote-llm
      </div>
      <div
        style={{
          position: 'fixed',
          bottom: 24,
          right: 24,
          fontSize: 10,
          letterSpacing: '0.2em',
          color: '#191919',
          textTransform: 'uppercase',
        }}
      >
        v1.0
      </div>

      <div
        className="animate-slide-up"
        style={{ width: 300 }}
      >
        {/* Wordmark */}
        <div style={{ marginBottom: 40, textAlign: 'center' }}>
          <div
            style={{
              fontSize: 11,
              letterSpacing: '0.45em',
              textTransform: 'uppercase',
              color: '#00E87A',
              fontWeight: 600,
              marginBottom: 8,
            }}
          >
            vote-llm
          </div>
          <div
            style={{
              width: 24,
              height: 1,
              background: '#191919',
              margin: '0 auto',
            }}
          />
          <div
            style={{
              marginTop: 8,
              fontSize: 10,
              letterSpacing: '0.2em',
              color: '#282420',
              textTransform: 'uppercase',
            }}
          >
            authentication required
          </div>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
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
              style={{
                width: '100%',
                padding: '10px 12px',
                background: '#0C0C0C',
                border: '1px solid #191919',
                color: '#C4C0AC',
                fontSize: 12,
                letterSpacing: '0.04em',
                outline: 'none',
                borderRadius: 0,
                boxSizing: 'border-box',
                transition: 'border-color 150ms',
              }}
              onFocus={(e) => (e.target.style.borderColor = '#302E2A')}
              onBlur={(e) => (e.target.style.borderColor = '#191919')}
            />
            {error && (
              <p
                style={{
                  marginTop: 6,
                  fontSize: 11,
                  color: '#FF3A3A',
                  letterSpacing: '0.04em',
                }}
              >
                {error}
              </p>
            )}
          </div>
          <button
            type="submit"
            style={{
              width: '100%',
              padding: '10px 12px',
              background: '#00E87A',
              color: '#070707',
              fontSize: 11,
              letterSpacing: '0.2em',
              textTransform: 'uppercase',
              fontWeight: 600,
              border: 'none',
              cursor: 'pointer',
              borderRadius: 0,
              transition: 'opacity 150ms',
            }}
            onMouseEnter={(e) => ((e.target as HTMLElement).style.opacity = '0.85')}
            onMouseLeave={(e) => ((e.target as HTMLElement).style.opacity = '1')}
          >
            Continue
          </button>
        </form>
      </div>
    </div>
  );
}
