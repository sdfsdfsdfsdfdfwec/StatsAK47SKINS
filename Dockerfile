# Build stage - Go backend
FROM golang:1.21-alpine AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/*.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# Build stage - React frontend
FROM node:20-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci --only=production
COPY frontend/ ./
RUN npm run build

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=backend /app/server .
COPY --from=frontend /app/build ./static
ENV GIN_MODE=release
ENV STATIC_DIR=/app/static
EXPOSE 8080
CMD ["./server"]
