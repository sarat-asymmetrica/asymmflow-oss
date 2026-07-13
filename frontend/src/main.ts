import './assets/design-tokens.css'
import './app.css'
import './assets/theme.css'
import './assets/layout.css'
import './assets/phi-token-compat.css' // Wave 11 A2: bridge phi token names → loaded semantic tokens
import App from './App.svelte'
import { LogError } from '../wailsjs/runtime/runtime'
import { mount } from "svelte";

function describeFrontendError(error: unknown): string {
  if (error instanceof Error) {
    return `${error.message}\n${error.stack || ''}`.trim()
  }
  return String(error)
}

function showBootError(error: unknown) {
  const target = document.getElementById('app') || document.body
  const message = describeFrontendError(error)
  try {
    LogError(`[frontend] ${message}`)
  } catch {
    console.error('[frontend]', error)
  }

  target.innerHTML = `
    <div style="
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 32px;
      background: #edf3f7;
      color: #1d1d1f;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    ">
      <section style="
        width: min(760px, 100%);
        background: #fff;
        border: 1px solid #d8e0e8;
        border-radius: 8px;
        padding: 24px;
        box-shadow: 0 18px 44px rgba(29, 29, 31, 0.08);
      ">
        <h1 style="margin: 0 0 10px; font-size: 22px;">Frontend render failed</h1>
        <p style="margin: 0 0 16px; color: #606b78;">The backend is alive, but the UI hit a startup exception.</p>
        <pre style="
          white-space: pre-wrap;
          word-break: break-word;
          margin: 0;
          padding: 14px;
          background: #f6f8fb;
          border: 1px solid #dfe7ef;
          border-radius: 6px;
          color: #293447;
          font-size: 12px;
          line-height: 1.5;
        ">${message.replace(/[&<>"']/g, (char) => ({
          '&': '&amp;',
          '<': '&lt;',
          '>': '&gt;',
          '"': '&quot;',
          "'": '&#39;',
        })[char] || char)}</pre>
      </section>
    </div>
  `
}

window.addEventListener('error', (event) => {
  showBootError(event.error || event.message)
})

window.addEventListener('unhandledrejection', (event) => {
  showBootError(event.reason)
})

const target = document.getElementById('app')
if (!target) {
  showBootError(new Error('Missing #app mount target'))
  throw new Error('Missing #app mount target')
}

let app: App
try {
  app = mount(App, { target })
} catch (error) {
  showBootError(error)
  throw error
}

export default app
