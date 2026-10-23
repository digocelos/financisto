import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import './styles.css'

function App() {
  const [apiStatus, setApiStatus] = useState('checking')

  useEffect(() => {
    fetch(`${import.meta.env.VITE_API_URL ?? 'http://localhost:8080'}/healthz`)
      .then((response) => {
        if (!response.ok) throw new Error('API unavailable')
        return response.json()
      })
      .then(() => setApiStatus('online'))
      .catch(() => setApiStatus('offline'))
  }, [])

  return (
    <main className="shell">
      <section className="card" aria-labelledby="title">
        <p className="eyebrow">Controle financeiro familiar</p>
        <h1 id="title">Financisto</h1>
        <p>Organize contas, objetivos e decisões financeiras da sua família.</p>
        <p className={`status status-${apiStatus}`} role="status">
          API: {apiStatus === 'checking' ? 'verificando…' : apiStatus === 'online' ? 'online' : 'indisponível'}
        </p>
      </section>
    </main>
  )
}

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
