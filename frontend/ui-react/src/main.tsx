import React from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { FluentProvider, webLightTheme } from '@fluentui/react-components'
import App from './App.tsx'
import { SkipLink } from './components/SkipLink.tsx'
import './index.css'

const rootEl = document.getElementById('root');
if (rootEl) {
  const root = createRoot(rootEl);
  root.render(
    <React.StrictMode>
      <FluentProvider theme={webLightTheme}>
        <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
          <SkipLink />
          <App />
        </BrowserRouter>
      </FluentProvider>
    </React.StrictMode>
  );
}
