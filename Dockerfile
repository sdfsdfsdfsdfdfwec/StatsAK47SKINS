# Build stage - Go backend
FROM golang:1.22-alpine AS backend
RUN apk add --no-cache git
WORKDIR /app
COPY backend/ ./
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# Build stage - React frontend
FROM node:20-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=backend /app/server .
COPY --from=frontend /app/build ./static
ENV STATIC_DIR=/app/static
ENV PORT=8080
EXPOSE 8080
CMD ["./server"]
