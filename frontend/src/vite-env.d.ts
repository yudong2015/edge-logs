/**
 * Vite environment variable type definitions
 */

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  readonly VITE_API_TIMEOUT: string
  readonly VITE_ENABLE_QUERY_HISTORY: string
  readonly VITE_ENABLE_AUTO_COMPLETE: string
  readonly VITE_DEFAULT_PAGE_SIZE: string
  readonly VITE_MAX_PAGE_SIZE: string
  readonly VITE_QUERY_DEBOUNCE_MS: string
  // Add more env variables here as needed
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
