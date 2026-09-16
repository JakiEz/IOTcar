# Multi-stage build: compile the Go binary in a full Go image, then ship only
# the binary in a tiny Alpine image (no Go toolchain, no source code).
FROM golang:1.25-alpine AS build
WORKDIR /src

# Copy dependency manifests first so `go mod download` is cached separately
# from source changes — editing a .go file won't re-download modules.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 = a fully static binary (needed for it to run on Alpine's musl libc)
RUN CGO_ENABLED=0 GOOS=linux go build -o /cmt-backend .

FROM alpine:3.20
# CA certs so the container can make TLS connections if it ever needs to
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /cmt-backend .

EXPOSE 8080
# Config (DB_DSN, PORT, SIMULATE) comes from docker-compose's `environment:` —
# no .env file is baked into or expected inside the image.
CMD ["./cmt-backend"]
