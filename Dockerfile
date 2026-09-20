# syntax=docker/dockerfile:1

FROM golang:1.24 AS gateway-builder
WORKDIR /src
COPY gateway/go.mod ./
COPY gateway/main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/sol-gateway .

FROM xpzouying/xiaohongshu-mcp:latest

COPY --from=gateway-builder /out/sol-gateway /app/sol-gateway
COPY start.sh /app/start.sh

RUN chmod +x /app/start.sh /app/sol-gateway && mkdir -p /data

ENV DATA_DIR=/data
ENV PORT=8080
ENV UPSTREAM_URL=http://127.0.0.1:18061

EXPOSE 8080

CMD ["/app/start.sh"]
