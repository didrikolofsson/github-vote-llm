import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import PortalPage from "./pages/portal/PortalPage";
import "./index.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, retry: 1 },
  },
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter basename="/portal.html">
        <Routes>
          <Route path="/:orgSlug/:repoName" element={<PortalPage />} />
          <Route
            path="*"
            element={
              <div className="min-h-screen bg-background flex items-center justify-center">
                <p className="text-muted-foreground text-sm">Portal not found.</p>
              </div>
            }
          />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
);
