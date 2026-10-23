import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import './styles.css'

function App() {
  const apiURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'
  const [apiStatus, setApiStatus] = useState('checking')
  const [session, setSession] = useState(null)
  const [familyName, setFamilyName] = useState('')
  const [members, setMembers] = useState([])
  const [memberForm, setMemberForm] = useState({ name: '', email: '' })
  const [catalogs, setCatalogs] = useState({ institutions: [], accountTypes: [] })
  const [accounts, setAccounts] = useState([])
  const [dashboard, setDashboard] = useState(null)
  const [accountForm, setAccountForm] = useState({ memberId: '', institutionId: '', accountTypeId: '', name: '', openingBalanceCents: 0 })
  const invitationToken = window.location.pathname.startsWith('/invite/') ? window.location.pathname.split('/')[2] : ''

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

  const loadMembers = (familyId) => fetch(`${apiURL}/v1/families/${familyId}/members`, {
    credentials: 'include', headers: { 'X-Family-ID': familyId },
  }).then((response) => response.ok ? response.json() : [])

  useEffect(() => {
    if (session?.activeFamilyId) {
      loadMembers(session.activeFamilyId).then((value) => {
        setMembers(value)
        if (value[0] && !accountForm.memberId) setAccountForm((current) => ({ ...current, memberId: value[0].id }))
      })
      fetch(`${apiURL}/v1/catalogs`, { credentials: 'include' }).then((response) => response.json()).then(setCatalogs)
      fetch(`${apiURL}/v1/families/${session.activeFamilyId}/accounts`, { credentials: 'include', headers: { 'X-Family-ID': session.activeFamilyId } }).then((response) => response.ok ? response.json() : []).then(setAccounts)
      fetch(`${apiURL}/v1/families/${session.activeFamilyId}/dashboard`, { credentials: 'include', headers: { 'X-Family-ID': session.activeFamilyId } }).then((response) => response.ok ? response.json() : null).then(setDashboard)
    }
  }, [session?.activeFamilyId])

  const startLogin = (event) => {
    event.preventDefault()
    const params = new URLSearchParams({ family_name: familyName.trim() })
    if (invitationToken) params.set('invitation_token', invitationToken)
    window.location.assign(`${apiURL}/auth/login?${params}`)
  }

  const switchFamily = (event) => {
    const familyId = event.target.value
    fetch(`${apiURL}/v1/me`, { credentials: 'include', headers: { 'X-Family-ID': familyId } })
      .then((response) => response.json()).then(setSession)
  }

  const addMember = (event) => {
    event.preventDefault()
    fetch(`${apiURL}/v1/families/${session.activeFamilyId}/members`, {
      method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', 'X-Family-ID': session.activeFamilyId },
      body: JSON.stringify(memberForm),
    }).then((response) => response.json()).then((member) => {
      setMembers((current) => [...current, member])
      setMemberForm({ name: '', email: '' })
    })
  }

  const inviteMember = (memberId) => {
    fetch(`${apiURL}/v1/families/${session.activeFamilyId}/members/${memberId}/invitation`, {
      method: 'POST', credentials: 'include', headers: { 'X-Family-ID': session.activeFamilyId },
    }).then((response) => response.json()).then((invitation) => navigator.clipboard?.writeText(invitation.invitationUrl))
  }

  const addAccount = (event) => {
    event.preventDefault()
    fetch(`${apiURL}/v1/families/${session.activeFamilyId}/accounts`, {
      method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', 'X-Family-ID': session.activeFamilyId },
      body: JSON.stringify({ ...accountForm, currency: 'BRL' }),
    }).then((response) => response.json()).then((account) => setAccounts((current) => [...current, account]))
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
            <label htmlFor="active-family">Família ativa</label>
            <select id="active-family" value={session.activeFamilyId} onChange={switchFamily}>
              {session.families.map((family) => <option key={family.id} value={family.id}>{family.name}</option>)}
            </select>
            {dashboard && <section className="dashboard" aria-label="Resumo financeiro">
              <div className="metrics"><strong>Saldo atual<br />R$ {(dashboard.currentBalanceCents / 100).toFixed(2)}</strong><strong>Receitas<br />R$ {(dashboard.incomeCents / 100).toFixed(2)}</strong><strong>Despesas<br />R$ {(dashboard.expenseCents / 100).toFixed(2)}</strong></div>
              <p>Saldo projetado: <strong>R$ {(dashboard.projectedBalanceCents / 100).toFixed(2)}</strong></p>
            </section>}
            <form className="onboarding" onSubmit={addMember}>
              <label htmlFor="member-name">Novo membro</label>
              <input id="member-name" value={memberForm.name} onChange={(event) => setMemberForm({ ...memberForm, name: event.target.value })} placeholder="Nome" required />
              <input type="email" value={memberForm.email} onChange={(event) => setMemberForm({ ...memberForm, email: event.target.value })} placeholder="E-mail" required />
              <button type="submit">Adicionar membro</button>
            </form>
            <ul className="members">
              {members.map((member) => <li key={member.id}>{member.name} <small>{member.email}</small>{!member.linked && <button type="button" onClick={() => inviteMember(member.id)}>Convidar</button>}</li>)}
            </ul>
            <form className="onboarding" onSubmit={addAccount}>
              <label htmlFor="account-name">Nova conta bancária</label>
              <input id="account-name" value={accountForm.name} onChange={(event) => setAccountForm({ ...accountForm, name: event.target.value })} placeholder="Conta principal" required />
              <select value={accountForm.memberId} onChange={(event) => setAccountForm({ ...accountForm, memberId: event.target.value })} required><option value="">Titular</option>{members.map((member) => <option key={member.id} value={member.id}>{member.name}</option>)}</select>
              <select value={accountForm.institutionId} onChange={(event) => setAccountForm({ ...accountForm, institutionId: event.target.value })} required><option value="">Instituição</option>{catalogs.institutions.map((institution) => <option key={institution.id} value={institution.id}>{institution.name}</option>)}</select>
              <select value={accountForm.accountTypeId} onChange={(event) => setAccountForm({ ...accountForm, accountTypeId: event.target.value })} required><option value="">Tipo de conta</option>{catalogs.accountTypes.map((type) => <option key={type.id} value={type.id}>{type.name}</option>)}</select>
              <input type="number" step="0.01" placeholder="Saldo inicial em reais" onChange={(event) => setAccountForm({ ...accountForm, openingBalanceCents: Math.round(Number(event.target.value || 0) * 100) })} />
              <button type="submit">Cadastrar conta</button>
            </form>
            <ul className="members">{accounts.map((account) => <li key={account.id}>{account.name} <small>{account.institution} · R$ {(account.openingBalanceCents / 100).toFixed(2)}</small></li>)}</ul>
            <a href={`${apiURL}/auth/logout`}>Sair</a>
          </section>
        ) : apiStatus === 'online' ? (
          <form className="onboarding" onSubmit={startLogin}>
            {!invitationToken && <><label htmlFor="family-name">Nome da família</label><input id="family-name" value={familyName} onChange={(event) => setFamilyName(event.target.value)} placeholder="Família Vasconcelos" required /></>}
            <button type="submit">{invitationToken ? 'Entrar para aceitar convite' : 'Entrar com Google'}</button>
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
