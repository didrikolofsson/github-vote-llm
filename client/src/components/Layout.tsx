import { Outlet } from 'react-router-dom';
import { useAuth } from '../lib/auth';
import { useQuery } from '@tanstack/react-query';
import { listMyOrganizations } from '@/lib/api';
import { LogOut } from 'lucide-react';
import { Button } from '@/components/ui/button';

export default function Layout() {
  const { logout } = useAuth();
  const { data: orgs = [] } = useQuery({
    queryKey: ['organizations'],
    queryFn: () => listMyOrganizations(),
  });
  const org = orgs[0];

  return (
    <div className="flex min-h-screen bg-background">
      {/* Sidebar */}
      <aside className="w-[240px] shrink-0 flex flex-col border-r border-border bg-sidebar">
        {/* Organization */}
        <div className="p-4 border-b border-border">
          <div className="flex items-center gap-3 py-2 px-3">
            <div className="size-9 rounded-[10px] bg-muted flex items-center justify-center shrink-0">
              <span className="text-sm font-semibold text-foreground">
                {org?.name?.charAt(0)?.toUpperCase() ?? 'v'}
              </span>
            </div>
            <div className="min-w-0 flex-1">
              <span className="block text-sm font-semibold text-foreground truncate">
                {org?.name ?? 'vote-llm'}
              </span>
              <span className="block text-xs text-muted-foreground truncate">
                Organization
              </span>
            </div>
          </div>
        </div>

        {/* User / Sign out */}
        <div className="flex-1" />
        <div className="p-3 border-t border-border">
          <Button
            variant="ghost"
            className="w-full justify-start gap-3 text-muted-foreground hover:bg-sidebar-accent"
            onClick={() => void logout()}
          >
            <LogOut data-icon="inline-start" />
            Sign out
          </Button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 flex flex-col min-w-0 bg-background">
        <div className="flex-1 p-8 max-w-[1280px] w-full mx-auto">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
