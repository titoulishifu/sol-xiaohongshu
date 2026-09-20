# Privacy Policy — sol-xiaohongshu

_Last updated: 2026-09-21_

## Scope

This repository is designed as a single-owner, self-hosted MCP gateway. The person operating the deployment is responsible for the deployment, its hosting account, and the Xiaohongshu account connected to it.

## Data processed

Depending on the MCP tools you invoke, the service may process:

- text queries and tool parameters selected by the AI client;
- Xiaohongshu note, profile, comment, engagement, and account-session data returned or used by the upstream MCP server;
- content you explicitly ask the tool to publish, comment, like, favorite, or otherwise act on;
- Xiaohongshu cookies and browser-profile data necessary to maintain the logged-in session.

## Storage

The gateway itself does not intentionally persist ChatGPT prompts or MCP request bodies.

Xiaohongshu cookies and browser-profile state are stored on the deployment's persistent disk so the Xiaohongshu login can survive restarts.

OAuth access and refresh tokens issued by this gateway are self-contained signed tokens and are not intentionally stored in a database by the gateway.

The hosting provider may retain infrastructure logs according to the provider's settings and policies.

## Purpose

Data is processed only to provide the MCP functions requested by the connected user, including reading Xiaohongshu content and, when explicitly invoked, performing supported account actions.

## Sharing

The deployment necessarily communicates with:

- the hosting provider chosen by the operator;
- the upstream `xpzouying/xiaohongshu-mcp` software running inside the same deployment;
- Xiaohongshu endpoints accessed by that upstream software;
- OpenAI products that the operator chooses to connect to the MCP endpoint.

This project does not intentionally sell user data.

## Retention and deletion

The operator controls retention of the persistent disk and hosting logs.

Deleting the persistent `/data` storage removes the locally persisted Xiaohongshu login/session files. Rotating `OAUTH_SIGNING_SECRET` invalidates previously issued gateway OAuth tokens.

## Security

Secrets should be stored only in the deployment platform's secret/environment-variable system and must not be committed to Git.

## Third-party services

This project is not affiliated with or endorsed by Xiaohongshu or OpenAI. Use of Xiaohongshu remains subject to Xiaohongshu's applicable terms and policies.

## Contact

For a public deployment, replace this section with a valid operator contact method before publishing or submitting the service for review.
