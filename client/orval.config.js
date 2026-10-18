import { defineConfig } from "orval";

export default defineConfig({
  api: {
    input: '../server/internal/api/api.yaml',
    output: {
      target: './src/api.ts',
      client: 'svelte-query',
      override: {
        mutator: {
          path: './src/custom-fetch.ts',
          name: 'customFetch',
        },
        query: {
          useQuery: true,
          useInfinite: true,
          useInfiniteQueryParam: 'nextId',
          options: {
            staleTime: 10000,
          }
        }
      }
    }
  }
})
