import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import BoardPage from './pages/BoardPage';
import './index.css';

const qc = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 10_000 } },
});

ReactDOM.createRoot(document.getElementById('board-root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={qc}>
      <BrowserRouter basename="/board">
        <Routes>
          <Route path="/:owner/:repo" element={<BoardPage />} />
          <Route path="/:owner/:repo/*" element={<BoardPage />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  </React.StrictMode>,
);
