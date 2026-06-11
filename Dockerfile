FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=1 go build -o hy2board .

FROM alpine:latest
RUN apk add --no-cache ca-certificates sqlite-libs tzdata
WORKDIR /app
COPY --from=backend /app/hy2board .
COPY --from=backend /app/web/dist ./web/dist
COPY config.yaml .
EXPOSE 9000
CMD ["./hy2board"]
