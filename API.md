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

---

# Management API

Grants full access to sending mails and managing lists. A cli client is provided. Third-party frontends should be reasonably easy to set up by using the following documentation.

All management endpoints require Ed25519 request signing when a public key is configured on the server.
This API should not be exposed publicly, even if it is authenticated. Prefer some kind of private tunneling/VPN, or using it right on the machine the server runs on via ssh.
The request signing is intended to be additional hardening, not the main security measure.

**Required headers**

| Header        | Value                                                                      |
|---------------|----------------------------------------------------------------------------|
| `X-Timestamp` | Unix timestamp (seconds) of the request                                    |
| `X-Signature` | Hex-encoded Ed25519 signature over `timestamp\nMETHOD\npath\nbodyHash`     |

The signed message is: `<timestamp>\n<METHOD>\n<path>\n<sha256(body) as hex>`

Requests whose timestamp differs from the server's clock by more than 5 minutes are rejected.

---

## POST `/lists`

Create a new mailing list.

**Body** — `application/json`

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
{ "id": 1, "name": "my-list" }
```

---

## GET `/lists/{id}`

Get list details including subscriber counts.

**Path params**
- `id` — numeric list ID

**Responses**

| Status | Meaning        |
|--------|----------------|
| `200`  | List details   |
| `400`  | Invalid ID     |
| `404`  | List not found |

**Response body**
```json
{
  "id": 1,
  "name": "my-list",
  "subscribers": { "total": 42, "confirmed": 38 }
}
```

---

## PUT `/lists/{id}`

Rename a mailing list.

**Path params**
- `id` — numeric list ID

**Body** — `application/json`

| Field  | Type   | Required |
|--------|--------|----------|
| `name` | string | yes      |

**Responses**

| Status | Meaning              |
|--------|----------------------|
| `200`  | Renamed list         |
| `400`  | Missing/invalid name |
| `500`  | Internal error       |

---

## DELETE `/lists/{id}`

Delete a mailing list and all its subscribers.

**Path params**
- `id` — numeric list ID

**Responses**

| Status | Meaning        |
|--------|----------------|
| `204`  | Deleted        |
| `400`  | Invalid ID     |
| `500`  | Internal error |

---

## GET `/lists/{id}/users`

List all subscribers of a mailing list.

**Path params**
- `id` — numeric list ID

**Responses**

| Status | Meaning         |
|--------|-----------------|
| `200`  | Subscriber list |
| `400`  | Invalid ID      |
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

Render a markdown newsletter and send it to all confirmed subscribers of the named list.

**Path params**
- `name` — list name

**Body** — `application/json`

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

## POST `/mail/test`

Send a rendered test mail to a single recipient without touching any list.

**Body** — `application/json`

| Field             | Type   | Required | Description             |
|-------------------|--------|----------|-------------------------|
| `recipient.name`  | string | no       | Recipient display name  |
| `recipient.email` | string | yes      | Recipient email address |
| `raw`             | string | yes      | Raw markdown content    |
| `data`            | object | no       | Template variables      |

**Responses**

| Status | Meaning                           |
|--------|-----------------------------------|
| `200`  | Test mail sent                    |
| `400`  | Missing `recipient.email` or `raw` |
| `500`  | Internal error                    |
