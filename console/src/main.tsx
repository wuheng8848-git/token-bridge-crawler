import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import React from 'react'
import ReactDOM from 'react-dom/client'

import { App } from './App'
import { AdminThemeProvider } from './theme/AdminThemeContext'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: false },
  },
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <AdminThemeProvider>
        <App />
      </AdminThemeProvider>
    </QueryClientProvider>
  </React.StrictMode>,
)
