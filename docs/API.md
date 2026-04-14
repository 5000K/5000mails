# Public API

All responses: `Content-Type: application/json`

---

## POST `/{listName}/subscribe`

Subscribe a user to the named mailing list. Triggers a double opt-in confirmation email.

**Path params**
- `listName` - name of the mailing list

**Body** - `application/json` or `application/x-www-form-urlencoded`

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

When redirect pages are configured, all outcomes issue a `303 See Other` instead of a JSON body. See [CONFIG.md](CONFIG.md#redirects).

---

## GET `/confirm/{token}`

Complete double opt-in using the token from the confirmation email.

**Path params**
- `token` - 64-char hex token

**Responses**

| Status | Meaning                          |
|--------|----------------------------------|
| `200`  | Subscription confirmed           |
| `400`  | Token invalid or already used    |

---

## GET `/unsubscribe/{token}`

Remove a subscriber using their per-subscription unsubscribe token (included in every newsletter).

**Path params**
- `token` - 64-char hex unsubscribe token (unique per subscription)

**Responses**

| Status | Meaning                          |
|--------|----------------------------------|
| `200`  | Unsubscribed                     |
| `400`  | Token invalid or not found       |

---

# Management API

Grants full access to sending mails and managing lists. A CLI client is provided. Third-party frontends should be reasonably easy to set up by using the following documentation.

All management endpoints require Ed25519 request signing when a public key is configured on the server.
This API should not be exposed publicly, even if it is authenticated. Prefer some kind of private tunneling/VPN, or using it right on the machine the server runs on via SSH.
The request signing is intended to be additional hardening, not the main security measure.

**Required headers**

| Header        | Value                                                                  |
|---------------|------------------------------------------------------------------------|
| `X-Timestamp` | Unix timestamp (seconds) of the request                                |
| `X-Signature` | Hex-encoded Ed25519 signature over `timestamp\nMETHOD\npath\nbodyHash` |

The signed message is: `<timestamp>\n<METHOD>\n<path>\n<sha256(body) as hex>`

Requests whose timestamp differs from the server's clock by more than 5 minutes are rejected.

---

## GET `/lists`

Return all mailing lists.

**Responses**

| Status | Meaning        |
|--------|----------------|
| `200`  | List of lists  |
| `500`  | Internal error |

**Response body**
```json
[{ "name": "newsletter" }, { "name": "announcements" }]
```

---

## POST `/lists`

Create a new mailing list.

**Body** - `application/json`

| Field  | Type   | Required |
|--------|--------|----------|
| `name` | string | yes      |

**Responses**

| Status | Meaning              |
|--------|----------------------|
| `201`  | List created         |
| `400`  | Missing/invalid name |
| `500`  | Internal error       |

**Response body**
```json
{ "name": "newsletter" }
```

---

## GET `/lists/{name}`

Get list details including subscriber counts.

**Path params**
- `name` - list name

**Responses**

| Status | Meaning        |
|--------|----------------|
| `200`  | List details   |
| `404`  | List not found |
| `500`  | Internal error |

**Response body**
```json
{
  "name": "newsletter",
  "subscribers": { "total": 42, "confirmed": 38 }
}
```

---

## PUT `/lists/{name}`

Rename a mailing list.

**Path params**
- `name` - current list name

**Body** - `application/json`

| Field  | Type   | Required |
|--------|--------|----------|
| `name` | string | yes      | new name |

**Responses**

| Status | Meaning              |
|--------|----------------------|
| `200`  | Renamed              |
| `400`  | Missing/invalid name |
| `500`  | Internal error       |

**Response body**
```json
{ "name": "new-name" }
```

---

## DELETE `/lists/{name}`

Delete a mailing list and all its subscribers.

**Path params**
- `name` - list name

**Responses**

| Status | Meaning        |
|--------|----------------|
| `204`  | Deleted        |
| `500`  | Internal error |

---

## GET `/lists/{name}/users`

List all subscribers of a mailing list (confirmed and unconfirmed).

**Path params**
- `name` - list name

**Responses**

| Status | Meaning         |
|--------|-----------------|
| `200`  | Subscriber list |
| `500`  | Internal error  |

**Response body**
```json
[
  { "id": 1, "name": "Alice", "email": "alice@example.com", "confirmed": true },
  { "id": 2, "name": "Bob",   "email": "bob@example.com",   "confirmed": false }
]
```

---

## POST `/lists/{name}/send`

Render a markdown newsletter and send it immediately to all confirmed subscribers of the named list. The sent mail is archived and retrievable via the newsletters endpoints.

**Path params**
- `name` - list name

**Body** - `application/json`

| Field  | Type   | Required | Description                                   |
|--------|--------|----------|-----------------------------------------------|
| `raw`  | string | yes      | Raw markdown content of the mail              |
| `data` | object | no       | Template variables injected into the markdown |

**Responses**

| Status | Meaning         |
|--------|-----------------|
| `200`  | Mail dispatched |
| `400`  | Missing `raw`   |
| `500`  | Internal error  |

---

## POST `/lists/{name}/schedule`

Schedule a markdown newsletter for future delivery to all confirmed subscribers of the named list.

**Path params**
- `name` - list name

**Body** - `application/json`

| Field         | Type    | Required | Description                              |
|---------------|---------|----------|------------------------------------------|
| `raw`         | string  | yes      | Raw markdown content of the mail         |
| `scheduledAt` | integer | yes      | Delivery time as a unix timestamp (UTC)  |

**Responses**

| Status | Meaning                          |
|--------|----------------------------------|
| `201`  | Scheduled mail created           |
| `400`  | Missing `raw` or `scheduledAt`   |
| `500`  | Internal error                   |

**Response body**
```json
{
  "id": 3,
  "mailingListName": "newsletter",
  "scheduledAt": 1776042000,
  "sentAt": null
}
```

---

## POST `/mail/test`

Send a rendered test mail to a single recipient without touching any list.

**Body** - `application/json`

| Field             | Type   | Required | Description             |
|-------------------|--------|----------|-------------------------|
| `recipient.name`  | string | no       | Recipient display name  |
| `recipient.email` | string | yes      | Recipient email address |
| `raw`             | string | yes      | Raw markdown content    |
| `data`            | object | no       | Template variables      |

**Responses**

| Status | Meaning                            |
|--------|------------------------------------|
| `200`  | Test mail sent                     |
| `400`  | Missing `recipient.email` or `raw` |
| `500`  | Internal error                     |

---

## GET `/newsletters`

List all archived sent newsletters (summary, no recipient list).

**Responses**

| Status | Meaning        |
|--------|----------------|
| `200`  | Newsletter list|
| `500`  | Internal error |

**Response body**
```json
[
  {
    "id": 1,
    "subject": "Issue #12",
    "senderName": "The Team",
    "sentAt": "2026-04-14T10:00:00Z",
    "mailingLists": ["newsletter"]
  }
]
```

---

## GET `/newsletters/{id}`

Get a single archived newsletter including full recipient list and raw markdown.

**Path params**
- `id` - numeric newsletter ID

**Responses**

| Status | Meaning             |
|--------|---------------------|
| `200`  | Newsletter detail   |
| `400`  | Invalid ID          |
| `404`  | Newsletter not found|

**Response body**
```json
{
  "id": 1,
  "subject": "Issue #12",
  "senderName": "The Team",
  "rawMarkdown": "# Issue #12\nHello...",
  "sentAt": "2026-04-14T10:00:00Z",
  "recipients": [
    { "id": 1, "name": "Alice", "email": "alice@example.com", "confirmed": true }
  ],
  "mailingLists": ["newsletter"]
}
```

---

## DELETE `/newsletters/{id}`

Delete an archived newsletter record. Does not affect any subscribers or scheduled mails.

**Path params**
- `id` - numeric newsletter ID

**Responses**

| Status | Meaning        |
|--------|----------------|
| `204`  | Deleted        |
| `400`  | Invalid ID     |
| `500`  | Internal error |

---

## GET `/scheduled`

List all scheduled mails (including already-sent ones).

**Responses**

| Status | Meaning               |
|--------|-----------------------|
| `200`  | Scheduled mail list   |
| `500`  | Internal error        |

**Response body**
```json
[
  {
    "id": 3,
    "mailingListName": "newsletter",
    "scheduledAt": 1776042000,
    "sentAt": null
  },
  {
    "id": 1,
    "mailingListName": "announcements",
    "scheduledAt": 1775000000,
    "sentAt": 1775000060
  }
]
```

`sentAt` is `null` for pending mails and a unix timestamp once delivered.

---

## GET `/scheduled/{id}`

Get a single scheduled mail.

**Path params**
- `id` - numeric scheduled mail ID

**Responses**

| Status | Meaning                    |
|--------|----------------------------|
| `200`  | Scheduled mail detail      |
| `400`  | Invalid ID                 |
| `404`  | Scheduled mail not found   |

**Response body**
```json
{
  "id": 3,
  "mailingListName": "newsletter",
  "scheduledAt": 1776042000,
  "sentAt": null
}
```

---

## DELETE `/scheduled/{id}`

Delete a scheduled mail. Has no effect if the mail has already been sent.

**Path params**
- `id` - numeric scheduled mail ID

**Responses**

| Status | Meaning        |
|--------|----------------|
| `204`  | Deleted        |
| `400`  | Invalid ID     |
| `500`  | Internal error |

---

## PUT `/scheduled/{id}/schedule`

Change the delivery time of a scheduled mail.

**Path params**
- `id` - numeric scheduled mail ID

**Body** - `application/json`

| Field         | Type    | Required | Description                             |
|---------------|---------|----------|-----------------------------------------|
| `scheduledAt` | integer | yes      | New delivery time as a unix timestamp (UTC) |

**Responses**

| Status | Meaning                     |
|--------|-----------------------------|
| `200`  | Updated scheduled mail      |
| `400`  | Missing/invalid `scheduledAt` |
| `500`  | Internal error              |

**Response body**
```json
{
  "id": 3,
  "mailingListName": "newsletter",
  "scheduledAt": 1776128400,
  "sentAt": null
}
```

---

## PUT `/scheduled/{id}/content`

Replace the markdown content of a scheduled mail.

**Path params**
- `id` - numeric scheduled mail ID

**Body** - `application/json`

| Field | Type   | Required | Description              |
|-------|--------|----------|--------------------------|
| `raw` | string | yes      | New raw markdown content |

**Responses**

| Status | Meaning                |
|--------|------------------------|
| `200`  | Updated scheduled mail |
| `400`  | Missing `raw`          |
| `500`  | Internal error         |

**Response body**
```json
{
  "id": 3,
  "mailingListName": "newsletter",
  "scheduledAt": 1776042000,
  "sentAt": null
}
```
