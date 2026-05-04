---
name: Notifications PRD location
description: Location of the notifications feature PRD
type: reference
---

Notifications feature PRD is at `docs/notifications-prd.md`.

Key design decisions:
- In-app only (Phase 1), push notifications for mobile (Phase 2)
- Types: like_recipe, like_article, comment_recipe, comment_article
- Only content author notified (not thread participants)
- No self-notifications
- Client formats display text (not backend)
- Auto-mark-all-read on screen open (POST /notifications/read_all)
- Background polling of /unread_count for badge
- No storage limit for MVP
- Metrics deferred post-launch
