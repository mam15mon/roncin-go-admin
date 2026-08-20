FROM node:24.14.1-bookworm-slim AS web-builder

WORKDIR /src
RUN corepack enable
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY web/package.json web/package.json
RUN pnpm install --frozen-lockfile
COPY web web
RUN pnpm --dir web build

FROM golang:1.25.7-bookworm AS server-builder

WORKDIR /src
COPY server/go.mod server/go.sum ./server/
RUN go -C server mod download
COPY server server
COPY --from=web-builder /src/web/dist ./server/internal/webassets/dist
RUN CGO_ENABLED=0 go -C server build -trimpath -ldflags="-s -w" -o /out/roncin-server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=server-builder /out/roncin-server /app/roncin-server
COPY server/configs /app/configs

EXPOSE 8000 9000
ENTRYPOINT ["/app/roncin-server"]
CMD ["-conf", "/app/configs"]
