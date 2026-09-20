# OpenAI connection and publication notes

This document separates two different goals:

## 1. Private/personal remote MCP connection

Once this service is deployed to a stable public HTTPS origin, the endpoint can be tested from ChatGPT Developer Mode:

```text
https://YOUR-DOMAIN.example/mcp
```

The gateway publishes MCP OAuth discovery metadata and uses authorization-code + PKCE.

For a personal deployment, the OAuth authorization page is protected by `OWNER_PASSWORD`. A successful authorization grants access to the single Xiaohongshu session stored in the deployment.

## 2. Public OpenAI plugin-directory publication

Do **not** assume that successful Developer Mode connection means the integration is eligible for publication.

OpenAI's current plugin guidelines require, among other things:

- a stable, publicly accessible HTTPS MCP endpoint;
- correct MCP tool metadata and annotations;
- appropriate authentication/authorization for private data and write actions;
- a public privacy policy;
- verified publisher identity;
- domain verification when requested;
- review/test materials and working test credentials when authentication is used;
- appropriate authorization for third-party integrations and compliance with the third party's terms.

OpenAI also states that it cannot approve plugins whose primary purpose is acting as an **unofficial connector** for a third-party service.

Because this project relies on an unofficial Xiaohongshu integration, treat it as a private/development connector unless you have obtained the authorization needed for a compliant public integration.

## Domain verification endpoint

If the OpenAI submission portal gives you a domain verification token, set:

```text
OPENAI_APPS_CHALLENGE=<exact token from the portal>
```

The gateway will return that exact token at:

```text
/.well-known/openai-apps-challenge
```

Do not leave a stale challenge token configured after verification unless the portal requires it.

## Production checklist

- [ ] Deploy to a stable HTTPS hostname.
- [ ] Attach persistent storage at `/data`.
- [ ] Set a strong `OWNER_PASSWORD`.
- [ ] Set a random `OAUTH_SIGNING_SECRET`.
- [ ] Set the final `PUBLIC_BASE_URL`.
- [ ] Confirm `/health` returns 200.
- [ ] Confirm OAuth metadata endpoints return the public HTTPS origin.
- [ ] Connect in ChatGPT Developer Mode.
- [ ] Run `get_login_qrcode` and sign in to the intended Xiaohongshu account.
- [ ] Confirm `check_login_status`.
- [ ] Test read-only tools before any write tool.
- [ ] Never expose upstream port 18061.
- [ ] Review Xiaohongshu terms/policies before using or distributing the integration.
- [ ] Do not submit publicly without appropriate third-party authorization.
