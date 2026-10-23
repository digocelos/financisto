# PostgreSQL como banco principal

O sistema usará PostgreSQL como banco principal porque o domínio exige consistência relacional entre Famílias, Membros, Contas, Operações, auditoria e projeções. Migrations versionadas e um PostgreSQL real em Docker serão a base do desenvolvimento local e dos testes de integração.
