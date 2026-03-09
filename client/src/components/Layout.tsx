import { NavLink, Outlet } from 'react-router-dom';
import { useAuth } from '../lib/auth';

const navLink = ({ isActive }: { isActive: boolean }) =>
  [
    'text-[11px] tracking-[0.12em] uppercase transition-colors duration-150',
    isActive ? 'text-[#C4C0AC]' : 'text-[#403C34] hover:text-[#6A6458]',
  ].join(' ');

export default function Layout() {
  const { logout } = useAuth();

  return (
    <div style={{ minHeight: '100vh', background: '#070707', color: '#C4C0AC' }}>
      <header
        style={{
          position: 'sticky',
          top: 0,
          zIndex: 10,
          background: '#070707',
          borderBottom: '1px solid #191919',
          height: 40,
          display: 'flex',
          alignItems: 'center',
          paddingLeft: 24,
          paddingRight: 24,
          justifyContent: 'space-between',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 32 }}>
          <span
            style={{
              color: '#00E87A',
              fontSize: 11,
              letterSpacing: '0.35em',
              textTransform: 'uppercase',
              fontWeight: 600,
            }}
          >
            vote-llm
          </span>
          <nav style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
            <NavLink to="/" end className={navLink}>
              Roadmap
            </NavLink>
            <NavLink to="/runs" className={navLink}>
              Runs
            </NavLink>
            <NavLink to="/config" className={navLink}>
              Config
            </NavLink>
          </nav>
        </div>
        <button
          onClick={logout}
          style={{
            fontSize: 11,
            letterSpacing: '0.1em',
            textTransform: 'uppercase',
            color: '#282420',
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            transition: 'color 150ms',
          }}
          onMouseEnter={(e) => ((e.target as HTMLElement).style.color = '#403C34')}
          onMouseLeave={(e) => ((e.target as HTMLElement).style.color = '#282420')}
        >
          Sign out
        </button>
      </header>
      <main style={{ padding: '32px 24px', maxWidth: 1040, margin: '0 auto' }}>
        <Outlet />
      </main>
    </div>
  );
}
