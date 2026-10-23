import { formatCents, operationStatuses, operationTypes } from './operations.js'

function operationAmount(operation) {
  return operation.entries?.[0]?.amountCents ?? 0
}

export default function Dashboard({ session, apiURL, dashboard, operations, accounts, operationForm, setOperationForm, onAddOperation, onDeleteOperation, onUpdateOperationStatus, onOpenSetup }) {
  const setField = (field) => (event) => setOperationForm((current) => ({ ...current, [field]: event.target.value }))
  const familyName = session.families.find((family) => family.id === session.activeFamilyId)?.name

  return (
    <main className="workspace">
      <header className="topbar">
        <div className="brand-inline"><span className="brand-dot">F</span><strong>Financisto</strong><nav className="main-nav" aria-label="Navegação principal"><a className="active" href="#dashboard">Dashboard</a><a href="#operations">Lançamentos</a><a href="#planning">Planejamento</a><a href="#accounts">Contas</a></nav></div>
        <div className="topbar-actions"><span className="family-name">{familyName}</span><span className="avatar">{session.user.name?.[0] ?? 'U'}</span><button className="link-button" type="button" onClick={onOpenSetup}>Configurar</button><a href={`${apiURL}/auth/logout`}>Sair</a></div>
      </header>
      <section className="workspace-content">
        <div className="welcome"><div><p className="eyebrow">Visão geral</p><h1>Olá, {session.user.name?.split(' ')[0] ?? 'por aqui'}.</h1><p>Acompanhe e registre o que está acontecendo com o dinheiro da família.</p></div></div>
        <section className="metrics" aria-label="Resumo financeiro">
          <article><span>Saldo atual</span><strong>{formatCents(dashboard?.currentBalanceCents)}</strong></article>
          <article><span>Receitas no período</span><strong className="positive">{formatCents(dashboard?.incomeCents)}</strong></article>
          <article><span>Despesas no período</span><strong className="negative">{formatCents(dashboard?.expenseCents)}</strong></article>
          <article><span>Saldo projetado</span><strong>{formatCents(dashboard?.projectedBalanceCents)}</strong></article>
        </section>
        <div className="ledger-layout" id="operations">
          <section className="surface">
            <div className="section-heading"><div><p className="eyebrow">Novo lançamento</p><h2>Registrar movimento</h2></div></div>
            <form className="operation-form" onSubmit={onAddOperation}>
              <div className="form-row"><label>Tipo<select value={operationForm.type} onChange={setField('type')}>{operationTypes.map((type) => <option key={type.value} value={type.value}>{type.label}</option>)}</select></label><label>Status<select value={operationForm.status} onChange={setField('status')}>{operationStatuses.map((status) => <option key={status.value} value={status.value}>{status.label}</option>)}</select></label></div>
              <label>Conta<select value={operationForm.accountId} onChange={setField('accountId')} required><option value="">Selecione a conta</option>{accounts.map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}</select></label>
              {operationForm.type === 'TRANSFER' && <label>Conta de destino<select value={operationForm.destinationAccountId} onChange={setField('destinationAccountId')} required><option value="">Selecione a conta</option>{accounts.filter((account) => account.id !== operationForm.accountId).map((account) => <option key={account.id} value={account.id}>{account.name}</option>)}</select></label>}
              <label>Descrição<input value={operationForm.description} onChange={setField('description')} placeholder="Ex.: Mercado" required /></label>
              <div className="form-row"><label>Valor<input inputMode="decimal" value={operationForm.amount} onChange={setField('amount')} placeholder="0,00" required /></label><label>Data<input type="date" value={operationForm.effectiveAt} onChange={setField('effectiveAt')} required /></label></div>
              <button className="primary-button" type="submit">Salvar lançamento</button>
            </form>
          </section>
          <section className="surface operations-surface">
            <div className="section-heading"><div><p className="eyebrow">Movimentações</p><h2>Últimos lançamentos</h2></div><span className="count-badge">{operations.length}</span></div>
            {operations.length === 0 ? <div className="empty-state"><p>Nenhum lançamento registrado.</p><small>Adicione sua primeira receita, despesa ou transferência.</small></div> : <ul className="operations-list">{operations.map((operation) => { const amount = operationAmount(operation); return <li key={operation.id}><div className={`operation-icon ${amount >= 0 ? 'income' : 'expense'}`}>{amount >= 0 ? '↑' : '↓'}</div><div className="operation-info"><strong>{operation.description}</strong><small>{new Date(operation.effectiveAt).toLocaleDateString('pt-BR')}</small></div><strong className={amount >= 0 ? 'positive' : 'negative'}>{formatCents(Math.abs(amount))}</strong><select className="operation-status" aria-label={`Status de ${operation.description}`} value={operation.status} onChange={(event) => onUpdateOperationStatus(operation, event.target.value)}>{operationStatuses.map((status) => <option key={status.value} value={status.value}>{status.label}</option>)}</select><button className="icon-button" type="button" aria-label={`Excluir ${operation.description}`} onClick={() => onDeleteOperation(operation)}>Excluir</button></li> })}</ul>}
          </section>
        </div>
      </section>
    </main>
  )
}
