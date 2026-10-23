import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { buildOperationPayload } from './operations.js'
import Dashboard from './Dashboard.jsx'
import Setup from './Setup.jsx'
import './styles.css'

function App() {
  const apiURL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'
  const [session, setSession] = useState(undefined)
  const [familyName, setFamilyName] = useState('')
  const [members, setMembers] = useState([])
  const [memberForm, setMemberForm] = useState({ name: '', email: '' })
  const [catalogs, setCatalogs] = useState({ institutions: [], accountTypes: [] })
  const [accounts, setAccounts] = useState([])
  const [dashboard, setDashboard] = useState(null)
  const [operations, setOperations] = useState([])
  const [operationForm, setOperationForm] = useState({ type: 'EXPENSE', status: 'CONFIRMED', accountId: '', destinationAccountId: '', amount: '', description: '', effectiveAt: new Date().toISOString().slice(0, 10), dueAt: '' })
  const [setupOpen, setSetupOpen] = useState(false)
  const [accountForm, setAccountForm] = useState({ memberId: '', institutionId: '', accountTypeId: '', name: '', openingBalanceCents: 0 })
  const invitationToken = window.location.pathname.startsWith('/invite/') ? window.location.pathname.split('/')[2] : ''

  useEffect(() => {
    fetch(`${apiURL}/v1/me`, { credentials: 'include' })
      .then((response) => {
        if (response.ok) return response.json()
        if (response.status === 401) return null
        throw new Error('Session unavailable')
      })
      .then(setSession)
      .catch(() => setSession(null))
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
      fetch(`${apiURL}/v1/families/${session.activeFamilyId}/operations`, { credentials: 'include', headers: { 'X-Family-ID': session.activeFamilyId } }).then((response) => response.ok ? response.json() : []).then(setOperations)
    }
  }, [session?.activeFamilyId])

  const startLogin = (event) => {
    event.preventDefault()
    const params = new URLSearchParams()
    if (invitationToken) params.set('invitation_token', invitationToken)
    window.location.assign(`${apiURL}/auth/login?${params}`)
  }

  const createFamily = (event) => {
    event.preventDefault()
    fetch(`${apiURL}/v1/families`, { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: familyName.trim() }) })
      .then((response) => response.ok ? fetch(`${apiURL}/v1/me`, { credentials: 'include' }) : response.json().then((error) => Promise.reject(error)))
      .then((response) => response.json()).then(setSession)
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

  const addOperation = (event) => {
    event.preventDefault()
    const familyID = session.activeFamilyId
    fetch(`${apiURL}/v1/families/${familyID}/operations`, {
      method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json', 'X-Family-ID': familyID },
      body: JSON.stringify(buildOperationPayload(operationForm)),
    }).then((response) => response.ok ? fetch(`${apiURL}/v1/families/${familyID}/operations`, { credentials: 'include', headers: { 'X-Family-ID': familyID } }) : response.json().then((error) => Promise.reject(error)))
      .then((response) => response.json()).then((value) => {
        setOperations(value)
        setOperationForm((current) => ({ ...current, amount: '', description: '' }))
      })
  }

  const deleteOperation = (operation) => {
    const familyID = session.activeFamilyId
    fetch(`${apiURL}/v1/families/${familyID}/operations/${operation.id}`, {
      method: 'DELETE', credentials: 'include', headers: { 'X-Family-ID': familyID, 'If-Match': String(operation.version) },
    }).then((response) => response.ok ? fetch(`${apiURL}/v1/families/${familyID}/operations`, { credentials: 'include', headers: { 'X-Family-ID': familyID } }) : Promise.reject(new Error('Não foi possível excluir o lançamento')))
      .then((response) => response.json()).then(setOperations)
  }

  const updateOperationStatus = (operation, status) => {
    const familyID = session.activeFamilyId
    fetch(`${apiURL}/v1/families/${familyID}/operations/${operation.id}`, {
      method: 'PATCH', credentials: 'include', headers: { 'Content-Type': 'application/json', 'X-Family-ID': familyID },
      body: JSON.stringify({ version: operation.version, status }),
    }).then((response) => response.ok ? fetch(`${apiURL}/v1/families/${familyID}/operations`, { credentials: 'include', headers: { 'X-Family-ID': familyID } }) : response.json().then((error) => Promise.reject(error)))
      .then((response) => response.json()).then(setOperations)
  }

  if (session === undefined) return null
  if (!session) return <main className="auth-shell"><section className="auth-panel"><div className="brand-mark">F</div><p className="eyebrow">Financisto</p><h1>Clareza para as decisões da sua família.</h1><p className="lead">Organize contas, acompanhe objetivos e planeje o próximo passo juntos.</p><button className="google-button" onClick={startLogin}>{invitationToken ? 'Entrar para aceitar convite' : 'Entrar com Google'}</button></section></main>
  if (session.families.length === 0) return <main className="auth-shell"><section className="auth-panel onboarding-panel"><div className="brand-mark">F</div><p className="eyebrow">Primeiro passo</p><h1>Vamos criar o espaço da sua família.</h1><p className="lead">Escolha um nome para começar a organizar suas finanças.</p><form className="onboarding" onSubmit={createFamily}><label htmlFor="family-name">Nome da família</label><input id="family-name" value={familyName} onChange={(event) => setFamilyName(event.target.value)} placeholder="Família Silva" required /><button className="primary-button" type="submit">Continuar</button></form></section></main>
  if (setupOpen) return <Setup apiURL={apiURL} session={session} members={members} accounts={accounts} catalogs={catalogs} accountForm={accountForm} setAccountForm={setAccountForm} onAddMember={addMember} memberForm={memberForm} setMemberForm={setMemberForm} onInviteMember={inviteMember} onAddAccount={addAccount} onClose={() => setSetupOpen(false)} />
  if (!setupOpen) return <Dashboard session={session} apiURL={apiURL} dashboard={dashboard} operations={operations} accounts={accounts} operationForm={operationForm} setOperationForm={setOperationForm} onAddOperation={addOperation} onDeleteOperation={deleteOperation} onUpdateOperationStatus={updateOperationStatus} onOpenSetup={() => setSetupOpen(true)} />
  if (!setupOpen) return <main className="workspace"><header className="topbar"><div className="brand-inline"><span className="brand-dot">F</span><strong>Financisto</strong></div><div className="topbar-actions"><select aria-label="Família ativa" value={session.activeFamilyId} onChange={switchFamily}>{session.families.map((family) => <option key={family.id} value={family.id}>{family.name}</option>)}</select><span className="avatar">{session.user.name?.[0] ?? 'U'}</span><a href={`${apiURL}/auth/logout`}>Sair</a></div></header><section className="workspace-content"><div className="welcome"><div><p className="eyebrow">Visão geral</p><h1>Olá, {session.user.name?.split(' ')[0] ?? 'por aqui'}.</h1><p>Acompanhe o que está acontecendo com o dinheiro da família.</p></div><button className="primary-button" type="button" onClick={() => setSetupOpen(true)}>Configurar família</button></div><section className="metrics" aria-label="Resumo financeiro"><article><span>Saldo atual</span><strong>R$ {((dashboard?.currentBalanceCents ?? 0) / 100).toFixed(2)}</strong></article><article><span>Receitas no período</span><strong className="positive">R$ {((dashboard?.incomeCents ?? 0) / 100).toFixed(2)}</strong></article><article><span>Despesas no período</span><strong className="negative">R$ {((dashboard?.expenseCents ?? 0) / 100).toFixed(2)}</strong></article><article><span>Saldo projetado</span><strong>R$ {((dashboard?.projectedBalanceCents ?? 0) / 100).toFixed(2)}</strong></article></section><section className="surface dashboard-empty"><p className="eyebrow">Tudo pronto para começar</p><h2>Seu painel financeiro está vazio.</h2><p>Adicione uma conta bancária e os membros da família para acompanhar saldos, receitas e despesas aqui.</p><button className="primary-button" type="button" onClick={() => setSetupOpen(true)}>Começar configuração</button></section></section></main>
  return <main className="workspace"><header className="topbar"><div className="brand-inline"><span className="brand-dot">F</span><strong>Financisto</strong></div><div className="topbar-actions"><select aria-label="Família ativa" value={session.activeFamilyId} onChange={switchFamily}>{session.families.map((family) => <option key={family.id} value={family.id}>{family.name}</option>)}</select><span className="avatar">{session.user.name?.[0] ?? 'U'}</span><a href={`${apiURL}/auth/logout`}>Sair</a></div></header><section className="workspace-content"><div className="welcome"><div><p className="eyebrow">Visão geral</p><h1>Olá, {session.user.name.split(' ')[0]}.</h1><p>Acompanhe o que está acontecendo com o dinheiro da família.</p></div></div>{dashboard && <section className="metrics" aria-label="Resumo financeiro"><article><span>Saldo atual</span><strong>R$ {(dashboard.currentBalanceCents / 100).toFixed(2)}</strong></article><article><span>Receitas no período</span><strong className="positive">R$ {(dashboard.incomeCents / 100).toFixed(2)}</strong></article><article><span>Despesas no período</span><strong className="negative">R$ {(dashboard.expenseCents / 100).toFixed(2)}</strong></article><article><span>Saldo projetado</span><strong>R$ {(dashboard.projectedBalanceCents / 100).toFixed(2)}</strong></article></section>}<div className="content-grid"><section className="surface"><div className="section-heading"><div><p className="eyebrow">Pessoas</p><h2>Membros da família</h2></div></div><form className="compact-form" onSubmit={addMember}><input aria-label="Nome do membro" value={memberForm.name} onChange={(event) => setMemberForm({ ...memberForm, name: event.target.value })} placeholder="Nome" required /><input aria-label="E-mail do membro" type="email" value={memberForm.email} onChange={(event) => setMemberForm({ ...memberForm, email: event.target.value })} placeholder="E-mail" required /><button className="primary-button" type="submit">Adicionar</button></form><ul className="members">{members.map((member) => <li key={member.id}><span className="list-avatar">{member.name[0]}</span><span><strong>{member.name}</strong><small>{member.email}</small></span>{!member.linked && <button type="button" onClick={() => inviteMember(member.id)}>Convidar</button>}</li>)}</ul></section><section className="surface"><div className="section-heading"><div><p className="eyebrow">Patrimônio</p><h2>Contas bancárias</h2></div></div><form className="compact-form account-form" onSubmit={addAccount}><input aria-label="Nome da conta" value={accountForm.name} onChange={(event) => setAccountForm({ ...accountForm, name: event.target.value })} placeholder="Nome da conta" required /><select aria-label="Titular" value={accountForm.memberId} onChange={(event) => setAccountForm({ ...accountForm, memberId: event.target.value })} required><option value="">Titular</option>{members.map((member) => <option key={member.id} value={member.id}>{member.name}</option>)}</select><select aria-label="Instituição" value={accountForm.institutionId} onChange={(event) => setAccountForm({ ...accountForm, institutionId: event.target.value })} required><option value="">Instituição</option>{catalogs.institutions.map((institution) => <option key={institution.id} value={institution.id}>{institution.name}</option>)}</select><select aria-label="Tipo" value={accountForm.accountTypeId} onChange={(event) => setAccountForm({ ...accountForm, accountTypeId: event.target.value })} required><option value="">Tipo</option>{catalogs.accountTypes.map((type) => <option key={type.id} value={type.id}>{type.name}</option>)}</select><input aria-label="Saldo inicial" type="number" step="0.01" placeholder="Saldo inicial" onChange={(event) => setAccountForm({ ...accountForm, openingBalanceCents: Math.round(Number(event.target.value || 0) * 100) })} /><button className="primary-button" type="submit">Cadastrar conta</button></form><ul className="members">{accounts.map((account) => <li key={account.id}><span className="list-avatar money">$</span><span><strong>{account.name}</strong><small>{account.institution}</small></span><strong>R$ {(account.openingBalanceCents / 100).toFixed(2)}</strong></li>)}</ul></section></div></section></main>
}

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
