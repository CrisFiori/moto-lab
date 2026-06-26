# --- Etapa 1: Compilación ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copiamos las definiciones de módulos primero para aprovechar la caché de Docker
COPY go.mod go.sum ./
RUN go mod download

# Copiamos el código fuente de tu PC
COPY main.go tools.go ./

# Compilamos el binario de forma estática (ideal para contenedores mínimos)
RUN CGO_ENABLED=0 GOOS=linux go build -o mcp-k8s-advisor .

# --- Etapa 2: Imagen Final de Producción ---
FROM alpine:3.19

WORKDIR /app

# Copiamos el binario desde la etapa de compilación
COPY --from=builder /app/mcp-k8s-advisor .

# Ejecutamos el servidor por defecto
CMD ["./mcp-k8s-advisor"]