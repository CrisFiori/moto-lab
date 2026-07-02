# === ETAPA 1: Compilación (Builder) ===
FROM golang:1.26-alpine AS builder

# Seteamos el directorio de trabajo dentro del contenedor de build
WORKDIR /app

# Copiamos los archivos de definición de dependencias
COPY go.mod go.sum ./

# Descargamos las librerías (Go las cachea en esta capa)
RUN go mod download

# Copiamos el resto del código fuente del dashboard
COPY . .

# Compilamos el binario estático puro para Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o cluster-dashboard .

# === ETAPA 2: Imagen de Producción (Minimalista) ===
FROM alpine:3.19

WORKDIR /app

# Copiamos el binario ejecutable desde la etapa anterior (builder)
COPY --from=builder /app/cluster-dashboard .

# CRUCIAL: Copiamos la carpeta del frontend para que el binario pueda leer el index.html
COPY --from=builder /app/ui ./ui

# Exponemos el puerto en el que escucha nuestra API de Go
EXPOSE 8080

# Comando por defecto para iniciar la aplicación (PID 1)
CMD ["./cluster-dashboard"]