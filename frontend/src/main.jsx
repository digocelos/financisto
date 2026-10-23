import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import './styles.css'

function App() {
  const apiURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'
  const [apiStatus, setApiStatus] = useState('checking')
  const [session, setSession] = useState(null)
  const [familyName, setFamilyName] = useState('')

  useEffect(() => {
    fetch(`${apiURL}/healthz`)
      .then((response) => {
        if (!response.ok) throw new Error('API unavailable')
        return response.json()
      })
      .then(() => {
        setApiStatus('online')
        return fetch(`${apiURL}/v1/me`, { credentials: 'include' })
      })
      .then((response) => {
        if (response.ok) return response.json()
        if (response.status === 401) return null
        throw new Error('Session unavailable')
      })
      .then(setSession)
      .catch(() => setApiStatus('offline'))
  }, [apiURL])

  const startLogin = (event) => {
    event.preventDefault()
    const params = new URLSearchParams({ family_name: familyName.trim() })
    window.location.assign(`${apiURL}/auth/login?${params}`)
  }

  return (
    <main className="shell">
      <section className="card" aria-labelledby="title">
        <p className="eyebrow">Controle financeiro familiar</p>
        <h1 id="title">Financisto</h1>
        <p>Organize contas, objetivos e decisões financeiras da sua família.</p>
        <p className={`status status-${apiStatus}`} role="status">
          API: {apiStatus === 'checking' ? 'verificando…' : apiStatus === 'online' ? 'online' : 'indisponível'}
        </p>
        {apiStatus === 'online' && session ? (
          <section className="session" aria-label="Sessão autenticada">
            <p>Olá, {session.user.name}.</p>
            <p>Família ativa: <strong>{session.families.find((family) => family.id === session.activeFamilyId)?.name}</strong></p>
            <a href={`${apiURL}/auth/logout`}>Sair</a>
          </section>
        ) : apiStatus === 'online' ? (
          <form className="onboarding" onSubmit={startLogin}>
            <label htmlFor="family-name">Nome da família</label>
            <input id="family-name" value={familyName} onChange={(event) => setFamilyName(event.target.value)} placeholder="Família Vasconcelos" required />
            <button type="submit">Entrar com Google</button>
          </form>
        ) : null}
      </section>
    </main>
  )
}

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
