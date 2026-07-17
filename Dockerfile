# syntax=docker/dockerfile:1

# ---- Build stage -----------------------------------------------------
# This stage has the full Go toolchain (~800MB+ with all its layers) and
# never ships — only the compiled binary it produces crosses into the
# runtime stage below. This is what "multi-stage build" means: two (or
# more) FROM statements in one Dockerfile, where later stages can COPY
# files out of earlier ones. The final image is built from the LAST
# stage only, so none of this stage's weight ends up in what you deploy.
FROM golang:1.26-alpine AS builder

# git is needed because `go mod download` fetches some dependencies
# directly from their VCS host rather than a proxy-served zip.
# ca-certificates is needed here too so `go mod download` can make
# outbound HTTPS calls against a proxy/VCS during the build.
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy only go.mod/go.sum first, then download — Docker caches each
# layer by the hash of what produced it. As long as go.mod/go.sum
# haven't changed, this layer (and the potentially slow `go mod
# download`) is reused from cache on every subsequent build, even
# after you've edited application code. Copying everything (COPY . .)
# before this step would invalidate the cache on every single source
# change, downloading every dependency from scratch every time.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 produces a statically linked binary with no dependency
# on the system's C library (glibc/musl). That's what makes it
# possible to run this binary on a minimal base image in the next
# stage — a CGO-enabled binary built against Alpine's musl libc would
# simply crash on a different base image, and vice versa.
#
# -ldflags="-s -w" strips debug symbols and the DWARF symbol table,
# shrinking the binary since this build will never be attached to with
# a Go debugger in production.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/server ./cmd/server

# ---- Runtime stage -----------------------------------------------------
# alpine (not scratch) is the base here deliberately: scratch has
# nothing in it at all — no shell, no package manager — which is the
# smallest possible image but makes `docker exec -it <container> sh`
# impossible when you need to debug a running container. Alpine adds
# ~5-7MB for that, which is worth it at this stage of the project.
FROM alpine:3.20

# ca-certificates is required at RUNTIME, not just build time: this
# app connects outbound to Postgres over TLS (Neon requires
# sslmode=require), and that TLS handshake needs a trusted CA bundle
# to verify the server's certificate. Without this, every database
# connection attempt fails with an x509 certificate error — an easy
# thing to forget since local development never hits this path (your
# host OS already has a CA bundle).
RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /app/bin/server .

# Never run a container process as root unless something specific
# requires it (binding to a port below 1024, for instance — 8080
# doesn't). If an attacker ever achieves code execution inside this
# container, running as an unprivileged user limits what that
# achieves; running as root hands them the whole container's
# filesystem and process namespace unrestricted.
USER app

EXPOSE 8080

ENTRYPOINT ["./server"]
