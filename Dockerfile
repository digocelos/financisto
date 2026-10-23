FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/financisto-api ./cmd/api

FROM alpine:3.22
RUN adduser -D -H -u 10001 app
USER app
COPY --from=build /out/financisto-api /financisto-api
EXPOSE 8080
ENTRYPOINT ["/financisto-api"]
