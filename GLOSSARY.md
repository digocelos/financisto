# Controle Financeiro Familiar

Vocabulário compartilhado do sistema de controle financeiro familiar. Os termos abaixo distinguem identidade, contexto familiar e operações financeiras para manter produto, API e interface consistentes.

## Identidade e escopo

**Usuário**:
Identidade autenticada pelo Google que pode acessar uma ou mais famílias.
_Evitar_: membro, pessoa, cliente

**Membro**:
Pessoa financeira pertencente a uma família, com ou sem Usuário vinculado.
_Evitar_: usuário familiar, conta

**Família**:
Contexto financeiro que reúne Membros, contas, cartões, lançamentos, objetivos e configurações próprias.
_Evitar_: grupo, organização

**Administrador global**:
Usuário interno da plataforma que mantém catálogos globais e pode bloquear Usuários ou Famílias.
_Evitar_: administrador da família

**Convite**:
Vínculo temporário enviado ao e-mail de um Membro para associar sua identidade Google à Família.
_Evitar_: solicitação, convite de usuário

## Catálogos e organização

**Catálogo global**:
Dados compartilhados pela plataforma, como instituições financeiras, bandeiras, tipos e moedas, mantidos pelo Administrador global.
_Evitar_: configuração da família

**Grupo de planejamento**:
Classificação ampla e personalizável pela Família que explica como uma categoria deve ser tratada no planejamento, como essencial, variável, discricionário, dívida, investimento ou objetivo.
_Evitar_: categoria principal

**Categoria**:
Classificação financeira pertencente à Família, organizada em até três níveis no MVP e com natureza de receita, despesa ou ambas.
_Evitar_: tipo de operação

**Tipo de operação**:
Tipo global predefinido que determina o comportamento financeiro de uma operação, como receita, despesa, transferência, aporte ou pagamento de cartão.
_Evitar_: categoria, grupo

**Subtipo de operação**:
Rótulo criado pela Família sob um Tipo de operação sem alterar seu comportamento financeiro.
_Evitar_: categoria

## Contas e dinheiro

**Conta bancária**:
Conta financeira de um Membro que recebe e movimenta dinheiro em uma instituição financeira.
_Evitar_: conta da família

**Cartão**:
Instrumento de crédito de um Membro com limite, fechamento, vencimento e faturas próprias.
_Evitar_: conta bancária, fatura

**Conta de investimento**:
Conta de um Membro usada para aportes, retiradas e rendimentos de investimentos, sem cotação automática no MVP.
_Evitar_: carteira, cofre

**Cofre**:
Reserva virtual alocada dentro de uma conta real para separar dinheiro destinado a um objetivo.
_Evitar_: conta bancária, investimento

**Objetivo**:
Meta financeira com valor-alvo, prazo, prioridade e plano de contribuições, podendo usar vários Cofres.
_Evitar_: orçamento, categoria

**Fatura**:
Conjunto de compras de um Cartão agrupadas por período de fechamento e com vencimento.
_Evitar_: lançamento de cartão

## Operações e planejamento

**Operação financeira**:
Ação financeira que pode gerar um ou mais Lançamentos relacionados, como transferência, pagamento de cartão ou aporte.
_Evitar_: lançamento isolado

**Lançamento**:
Registro financeiro individual com valor, datas, estado, origem, destino ou classificação, que altera o saldo de uma conta.
_Evitar_: transação quando o registro é individual

**Lançamento previsto**:
Lançamento esperado no futuro, ainda não confirmado, criado manualmente ou por uma Recorrência.
_Evitar_: pendência

**Recorrência**:
Regra que gera Lançamentos previstos para eventos repetidos, mantendo cada ocorrência editável de forma independente.
_Evitar_: parcela

**Parcela**:
Lançamento derivado de uma compra parcelada, vinculado à compra original e a uma Fatura específica.
_Evitar_: recorrência

**Conciliação**:
Vinculação revisada entre um Lançamento previsto e um registro importado ou realizado, podendo registrar desconto, multa ou pagamento parcial.
_Evitar_: importação automática

**Snapshot de saldo**:
Saldo anterior e saldo após um Lançamento, materializados para consulta rápida e recalculados quando alterações retroativas exigirem.
_Evitar_: saldo calculado apenas na tela

**Projeção**:
Estimativa futura que combina receitas, despesas, Recorrências, Parcelas, Faturas, investimentos, Cofres, Objetivos e reserva mínima.
_Evitar_: saldo atual
