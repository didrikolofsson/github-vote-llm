import { NavLink, Outlet } from 'react-router-dom';
import { useAuth } from '../lib/auth';

const navLink = ({ isActive }: { isActive: boolean }) =>
  [
    'text-[11px] tracking-[0.12em] uppercase transition-colors duration-150',
    isActive ? 'text-gray-100' : 'text-gray-400 hover:text-gray-300',
  ].join(' ');

export default function Layout() {
  const { logout } = useAuth();

  return (
    <div className="min-h-screen bg-gray-950 text-gray-100">
      <header className="sticky top-0 z-10 bg-gray-950 border-b border-gray-800 h-10 flex items-center px-6 justify-between">
        <div className="flex items-center gap-8">
          <span className="text-emerald-400 text-[11px] tracking-[0.35em] uppercase font-semibold">
            vote-llm
          </span>
          <nav className="flex items-center gap-6">
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
          onClick={() => void logout()}
          className="text-[11px] tracking-[0.1em] uppercase text-gray-500 hover:text-gray-400 bg-transparent border-none cursor-pointer transition-colors duration-150"
        >
          Sign out
        </button>
      </header>
      <main className="py-8 px-6 max-w-[1040px] mx-auto">
        <Outlet />
      </main>
    </div>
  );
}
