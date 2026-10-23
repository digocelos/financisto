## Agent skills

### Issue tracker

Issues e specs vivem no GitHub Issues de `digocelos/financisto`. Veja `docs/agents/issue-tracker.md`.

### Triage labels

Usar: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human` e `wontfix`. Veja `docs/agents/triage-labels.md`.

### Domain docs

Repositório single-context: `GLOSSARY.md` na raiz e decisões em `docs/adr/`. Veja `docs/agents/domain.md`.

## Pull requests

Cada ticket deve ser implementado em uma branch própria e entregue por uma Pull Request vinculada ao ticket.

- O autor da PR e dos commits deve ser `digocelos`.
- Os commits devem ser assinados com a chave GPG configurada no Git.
- Não adicionar trailers `Co-authored-by` aos commits.
- Executar os testes e validações do ticket antes de abrir a PR.
- A PR deve mencionar o ticket e resumir comportamento entregue, validações executadas e riscos conhecidos.
- Não aplicar mudanças diretamente em `main` quando o trabalho corresponder a um ticket.
