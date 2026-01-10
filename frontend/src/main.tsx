/**
 * Application Entry Point
 * Initializes React application with root rendering
 */

import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'

// Initialize React 18 root rendering
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
