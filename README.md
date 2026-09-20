# sol-xiaohongshu

A cloud-deployable, single-owner MCP wrapper around [xpzouying/xiaohongshu-mcp](https://github.com/xpzouying/xiaohongshu-mcp).

This repository does **not** reimplement Xiaohongshu access. It runs the upstream MCP server inside the same container and exposes it through a small OAuth 2.1 gateway suitable for connecting from ChatGPT Developer Mode.

## What this repo adds

- Public Streamable HTTP endpoint at `/mcp`
- OAuth 2.1 authorization-code + PKCE flow for ChatGPT
- Persistent Xiaohongshu cookies via `/data`
- A Render Blueprint for cloud deployment
- OpenAI domain-verification challenge endpoint
- Health check and privacy page
- Clear separation between the public OAuth token and the private upstream MCP bearer token

## Important limitation

This is an **unofficial** integration with Xiaohongshu. It is intended for personal/private use and development.

OpenAI's current public plugin guidelines state that apps whose primary purpose is to act as an unofficial connector for a third-party service are not eligible for public-directory approval, and third-party scraping/integration requires appropriate authorization and compliance with that service's terms.

That means:

- **Private ChatGPT Developer Mode connection:** technically supported after you deploy a public HTTPS endpoint.
- **Public OpenAI plugin-directory publication:** do not submit this as-is. You would first need appropriate authorization from Xiaohongshu and would need to satisfy OpenAI's publication requirements.

See [OPENAI.md](./OPENAI.md).

## Architecture

```text
ChatGPT / Codex
      |
      | HTTPS + OAuth access token
      v
sol-xiaohongshu gateway :$PORT
      |
      | private bearer token
      v
xpzouying/xiaohongshu-mcp :18061
      |
      v
Xiaohongshu web session
```

The upstream server stores its cookies and browser profile under `/data`. Use a persistent disk in production.

## Deploy on Render

This repo includes `render.yaml`.

1. Fork/use this repository in your Render account.
2. Create a Blueprint from the repository.
3. Set the required secret environment variables:
   - `OWNER_PASSWORD`: password you will enter when ChatGPT opens the OAuth authorization page.
   - `OAUTH_SIGNING_SECRET`: long random secret (recommend at least 32 random bytes).
4. Deploy.
5. After Render gives you a URL, set:
   - `PUBLIC_BASE_URL=https://YOUR-SERVICE.onrender.com`
6. Redeploy once after setting the final base URL.

The MCP endpoint will be:

```text
https://YOUR-SERVICE.onrender.com/mcp
```

The service also exposes:

- `/health`
- `/privacy`
- `/.well-known/oauth-protected-resource`
- `/.well-known/oauth-authorization-server`
- `/.well-known/openai-apps-challenge`

## Connect from ChatGPT Developer Mode

After deployment:

1. Enable Developer Mode in ChatGPT.
2. Create a custom MCP/plugin connection.
3. Use your deployed `https://.../mcp` URL.
4. ChatGPT should discover the OAuth metadata automatically.
5. When the authorization page opens, enter your `OWNER_PASSWORD`.
6. After connection, call `get_login_qrcode` and scan the Xiaohongshu QR code if the upstream session is not logged in.
7. Confirm with `check_login_status`.

## Local Docker test

Create an env file:

```bash
cp .env.example .env
```

Then fill in the secrets and run:

```bash
docker compose up --build
```

Local MCP URL:

```text
http://localhost:8080/mcp
```

For ChatGPT itself, localhost is not a production remote MCP endpoint. Use a real HTTPS deployment or OpenAI's supported secure MCP tunnel for private testing.

## Environment variables

| Variable | Required | Purpose |
| --- | --- | --- |
| `OWNER_PASSWORD` | yes | Single-owner OAuth login password |
| `OAUTH_SIGNING_SECRET` | yes | Signs OAuth codes and tokens |
| `PUBLIC_BASE_URL` | production | Canonical HTTPS origin, e.g. `https://mcp.example.com` |
| `PORT` | no | Public gateway port; defaults to `8080` |
| `DATA_DIR` | no | Persistent data dir; defaults to `/data` |
| `UPSTREAM_AUTH_TOKEN` | no | Private bearer token between gateway and upstream; generated per boot if omitted |
| `XHS_PROXY` | no | Proxy forwarded to upstream xiaohongshu-mcp |
| `OPENAI_APPS_CHALLENGE` | only for review | Exact OpenAI domain-verification token |

## Security notes

- Never expose upstream port `18061` publicly.
- Never commit `OWNER_PASSWORD`, `OAUTH_SIGNING_SECRET`, cookies, or access tokens.
- The OAuth gateway is intentionally single-owner. It is not a multi-tenant account system.
- Anyone who knows `OWNER_PASSWORD` can authorize access to the same Xiaohongshu session.
- Use a persistent disk so cookies survive redeploys.
- Rotate `OWNER_PASSWORD` and `OAUTH_SIGNING_SECRET` if they are ever leaked.
- A change to `OAUTH_SIGNING_SECRET` invalidates existing OAuth access/refresh tokens.

## Upstream

Runtime functionality is provided by:

- https://github.com/xpzouying/xiaohongshu-mcp
- Docker image: `xpzouying/xiaohongshu-mcp`

The upstream project is licensed under Apache-2.0. See [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md).

## License

The wrapper/gateway code in this repository is MIT licensed. The upstream image and upstream project remain under their own license.
