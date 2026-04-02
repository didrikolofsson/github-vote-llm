import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import {
  MutationCache,
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from '@tanstack/react-query';
import { z } from 'zod';
import { AuthProvider } from './lib/auth';
import { TooltipProvider } from './components/ui/tooltip';
import { createLogger } from './lib/logger';
import './index.css';
import App from './App';

const logger = createLogger('query');

const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) => {
      logger.error(`Query failed: ${query.queryKey.join(' > ')}`, error);
    },
  }),
  mutationCache: new MutationCache({
    onError: (error, _variables, _context, mutation) => {
      logger.error(`Mutation failed: ${mutation.mutationId}`, error);
    },
  }),
  defaultOptions: {
    queries: {
      // Don't retry schema validation errors — they won't resolve on retry.
      retry: (count, err) => !(err instanceof z.ZodError) && count < 1,
      staleTime: 10_000,
      refetchOnWindowFocus: false,
    },
  },
});

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <TooltipProvider>
          <App />
        </TooltipProvider>
      </AuthProvider>
    </QueryClientProvider>
  </StrictMode>,
);
