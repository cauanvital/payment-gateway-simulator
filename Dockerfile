# syntax=docker/dockerfile:1

# --- Stage de build ---
FROM golang:1.24-alpine AS build

WORKDIR /src

# Baixa dependências primeiro para aproveitar o cache de camadas.
COPY go.mod go.sum ./
RUN go mod download

# Copia o restante do código e compila um binário estático.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- Stage final (imagem mínima) ---
FROM gcr.io/distroless/static-debian12:nonroot AS final

WORKDIR /app
COPY --from=build /out/server /app/server

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
