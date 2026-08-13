FROM golang:1.24-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# distroless/static has no shell, no package manager, and runs as a non-root
# user by default (nonroot:nonroot) — this worker holds notification-provider
# credentials and handles household PII, so keeping the attack surface to
# "just the binary" is deliberate.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /out/worker /worker

EXPOSE 8090
ENTRYPOINT ["/worker"]
