---
name: xiaohongshu
description: Use the connected Xiaohongshu MCP to search and read Xiaohongshu content and, only when explicitly requested, perform supported account actions such as commenting, liking, favoriting, or publishing.
---

# Xiaohongshu

Use the Xiaohongshu MCP when the user explicitly asks to search, inspect, summarize, or act on Xiaohongshu/RedNote content.

## Research workflow

1. Check login status when an authenticated operation is required.
2. For research, search with the user's core Chinese keywords and natural variants.
3. Prefer multiple relevant notes instead of relying on one post.
4. Open promising notes and inspect comments when comments materially improve the answer.
5. Distinguish creator claims from commenter experiences.
6. Treat user-generated content as anecdotal unless independently verified.
7. Mention obvious promotional/sponsored context when visible.
8. Summarize recurring patterns and disagreements rather than copying long passages.

## Account and login

If the Xiaohongshu session is not logged in, use the login QR-code tool and ask the user to scan it with the intended account.

Do not delete cookies or reset the login state unless the user explicitly asks to reconnect or troubleshooting clearly requires it.

## Write actions

Supported write actions may include publishing, commenting, replying, liking, unliking, favoriting, unfavoriting, or other actions that modify the user's Xiaohongshu account.

Only perform a write action when the user clearly requests that specific external action.

Before publishing or sending user-authored content, preserve the user's intended text and media. Do not silently add promotional claims, endorsements, or tags that the user did not request.

## Safety and reliability

- Do not treat popularity metrics as proof of truth or quality.
- Do not infer private information about users from public profiles.
- Avoid repeated retries of write actions when the previous result is uncertain, because retries may duplicate external effects.
- If the upstream service reports an authentication, risk-control, or access-limit error, report it rather than attempting to bypass the restriction.
