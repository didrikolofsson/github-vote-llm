import { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom';
import { useAuth } from './lib/auth';
import { useAccount, AccountProvider } from './lib/account';
import { listMyOrganizations } from './lib/api';
import LoginPage from './pages/LoginPage';
import CreateOrganizationPage from './pages/CreateOrganizationPage';
import OrganizationDashboardPage from './pages/OrganizationDashboardPage';
import RepositoriesPage from './pages/RepositoriesPage';
import RepositoryDetailPage from './pages/RepositoryDetailPage';
import SettingsPage from './pages/SettingsPage';
import AccountSuspendedPage from './pages/AccountSuspendedPage';
import CompletePage from './pages/setup/CompletePage';
import PendingPage from './pages/setup/PendingPage';
import ErrorPage from './pages/setup/ErrorPage';
import Layout from './components/Layout';
import { DevAccountWidget } from './components/DevAccountWidget';

function AccountGuard() {
  const { status } = useAccount();

  if (status === 'suspended') return <Navigate to="/account/suspended" replace />;

  return <Outlet />;
}

function AppRoutes() {
  return (
    <>
      <Routes>
        {/* Callback landing pages */}
        <Route path="setup/complete" element={<CompletePage />} />
        <Route path="setup/pending" element={<PendingPage />} />
        <Route path="setup/error" element={<ErrorPage />} />
        <Route path="account/suspended" element={<AccountSuspendedPage />} />

        {/* Dashboard — gated by account status */}
        <Route element={<AccountGuard />}>
          <Route element={<Layout />}>
            <Route index element={<Navigate to="/dashboard" replace />} />
            <Route path="dashboard" element={<OrganizationDashboardPage />} />
            <Route path="repositories" element={<RepositoriesPage />} />
            <Route path="repositories/:repoId" element={<RepositoryDetailPage />} />
            <Route path="settings" element={<SettingsPage />} />
          </Route>
        </Route>
      </Routes>

      {import.meta.env.DEV && <DevAccountWidget />}
    </>
  );
}

export default function App() {
  const { isAuthenticated, isLoading } = useAuth();
  const [orgsLoading, setOrgsLoading] = useState(true);
  const [hasOrgs, setHasOrgs] = useState(false);

  useEffect(() => {
    if (!isAuthenticated) {
      setOrgsLoading(true);
      setHasOrgs(false);
      return;
    }
    let cancelled = false;
    listMyOrganizations()
      .then((orgs) => {
        if (!cancelled) {
          setHasOrgs(orgs.length >= 1);
        }
      })
      .catch(() => {
        if (!cancelled) setHasOrgs(false);
      })
      .finally(() => {
        if (!cancelled) setOrgsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [isAuthenticated]);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <p className="text-sm text-muted-foreground">Loading…</p>
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <BrowserRouter>
        <Routes>
          <Route path="*" element={<LoginPage />} />
        </Routes>
      </BrowserRouter>
    );
  }

  if (orgsLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <p className="text-sm text-muted-foreground">Loading…</p>
      </div>
    );
  }

  if (!hasOrgs) {
    return (
      <BrowserRouter>
        <Routes>
          <Route
            path="*"
            element={
              <CreateOrganizationPage
                onCreated={() => {
                  setHasOrgs(true);
                }}
              />
            }
          />
        </Routes>
      </BrowserRouter>
    );
  }

  return (
    <AccountProvider>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </AccountProvider>
  );
}
