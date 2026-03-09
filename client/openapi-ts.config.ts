import { defineConfig } from '@hey-api/openapi-ts';

export default defineConfig({
  input: '../server/openapi.yaml',
  output: {
    format: 'prettier',
    path: 'src/client',
  },
  plugins: [
    '@hey-api/client-fetch',
    '@hey-api/typescript',
    '@hey-api/sdk',
  ],
});
