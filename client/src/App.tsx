import { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from './lib/auth';
import { listMyOrganizations } from './lib/api';
import LoginPage from './pages/LoginPage';
import CreateOrganizationPage from './pages/CreateOrganizationPage';
import OrganizationDashboardPage from './pages/OrganizationDashboardPage';
import SettingsPage from './pages/SettingsPage';
import Layout from './components/Layout';

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
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<OrganizationDashboardPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
