# Public API

All responses: `Content-Type: application/json`

---

## POST `/{listName}/subscribe`

Subscribe a user to the named mailing list. Triggers a double opt-in confirmation email.

**Path params**
- `listName` — name of the mailing list

**Body** — `application/json` or `application/x-www-form-urlencoded`

| Field   | Type   | Required |
|---------|--------|----------|
| `name`  | string | yes      |
| `email` | string | yes      |

**Responses**

| Status | Meaning                |
|--------|------------------------|
| `202`  | Confirmation email sent |
| `400`  | Missing/invalid fields  |
| `500`  | Internal error          |

---

## GET `/confirm/{token}`

Complete double opt-in using the token from the confirmation email.

**Path params**
- `token` — 64-char hex token

**Responses**

| Status | Meaning                          |
|--------|----------------------------------|
| `200`  | Subscription confirmed           |
| `400`  | Token invalid or already used    |

---

## GET `/unsubscribe/{token}`

Remove a subscriber using their per-subscription unsubscribe token (included in every newsletter).

**Path params**
- `token` — 64-char hex unsubscribe token (unique per subscription)

**Responses**

| Status | Meaning                          |
|--------|----------------------------------|
| `200`  | Unsubscribed                     |
| `400`  | Token invalid or not found       |
