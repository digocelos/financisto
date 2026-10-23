# Financisto

Aplicativo de controle financeiro familiar.

## Ambiente local

Requisitos: Docker com Compose, Go 1.27+ e Node.js 22+.

Suba toda a fundação local:

```bash
docker compose up --build
```

Serviços:

- Frontend: <http://localhost:5173>
- API: <http://localhost:8080>
- PostgreSQL: `localhost:5432`
- Mock OIDC: <http://localhost:8081/default/.well-known/openid-configuration>

O ambiente local usa `ghcr.io/navikt/mock-oauth2-server`. OIDC é configurado por variáveis de ambiente (`OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET` e `OIDC_REDIRECT_URL`), permitindo trocar apenas a configuração para Google em produção.

Verificações rápidas:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

Testes:

```bash
go test ./...
go vet ./...
cd frontend && npm run build
```
