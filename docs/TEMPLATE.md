# Template Variables Reference

5000mails uses [Go templates](https://pkg.go.dev/text/template) in two places during the render pipeline:

1. **Markdown content** — the raw `.md` source (confirm-mail template or newsletter body) is executed as a Go template before Markdown parsing.
2. **HTML layout template** — the `template.html` wrapper is executed after Markdown-to-HTML conversion.

Both stages share the same data map, which is populated automatically with the variables below and merged with any custom `data` fields supplied via the API.

---

## Automatic variables

The table shows which variables are automatically injected in each sending context. Custom variables passed via `data` are always available on top of these.

| Variable                     | Type          | Confirm mail | Mail to list | Test mail | Description                                                                  |
| ---------------------------- | ------------- | :----------: | :----------: | :-------: | ---------------------------------------------------------------------------- |
| `Recipient`                  | `domain.User` |      ✓       |      ✓       |     ✓     | The recipient of this mail (see fields below)                                |
| `Recipient.ID`               | `uint`        |      ✓       |      ✓       |     ✓     | Database ID of the subscriber                                                |
| `Recipient.Name`             | `string`      |      ✓       |      ✓       |     ✓     | Display name                                                                 |
| `Recipient.Email`            | `string`      |      ✓       |      ✓       |     ✓     | Email address                                                                |
| `Recipient.MailingListName`  | `string`      |      ✓       |      ✓       |     ✓     | Name of the mailing list the subscriber belongs to                           |
| `Recipient.UnsubscribeToken` | `string`      |      ✓       |      ✓       |     ✓     | Opaque token used to build unsubscribe links                                 |
| `Recipient.ConfirmedAt`      | `*time.Time`  |      ✗¹      |      ✓       |     ✗     | Timestamp of double opt-in confirmation (`nil` if unconfirmed)               |
| `confirmURL`                 | `string`      |      ✓       |      ✗       |     ✗     | Full URL the subscriber must visit to confirm (`baseURL/confirm/<token>`)    |
| `token`                      | `string`      |      ✓       |      ✗       |     ✗     | Raw confirmation token (same value as the last path segment of `confirmURL`) |
| `unsubscribeURL`             | `string`      |      ✗       |      ✓       |     ✗     | Full URL to unsubscribe (`baseURL/unsubscribe/<UnsubscribeToken>`)           |

> ¹ Always `nil` in the confirmation mail — the user has not confirmed yet.

---

## HTML layout template variables

In addition to all variables above (and any custom `data`), the following keys are injected exclusively when the HTML layout template (`template.html`) is executed:

| Variable              | Type                  | Description                                                                                     |
| --------------------- | --------------------- | ----------------------------------------------------------------------------------------------- |
| `html`                | `string`              | Rendered HTML produced from the Markdown body                                                   |
| `metadata`            | `domain.MailMetadata` | Typed, parsed frontmatter (see fields below)                                                    |
| `metadata.Subject`    | `string`              | Email subject from the `subject` frontmatter field                                              |
| `metadata.SenderName` | `string`              | Sender display name from the `sender` frontmatter field                                         |
| `frontmatter`         | `map[string]any`      | Raw key-value map of **all** frontmatter fields, including any custom ones (e.g. `{{.frontmatter.myField}}`) |
| `theme`               | `string`              | Full CSS content loaded from the configured `theme` path                                        |

---

## Frontmatter

Every markdown template (confirm-mail and newsletter bodies) can include a YAML frontmatter block at the top. The renderer strips and parses it before Markdown processing — it is not rendered into the email body.

The known fields (`subject`, `sender`) are mapped into the typed `metadata` object. **All fields**, including any custom ones, are also available as a raw map under `frontmatter` in the HTML layout template.

```markdown
---
subject: "Your subject line"
sender: "Your Newsletter Name"
---

Body starts here…
```

| Field        | Description                                                                               |
|--------------|-------------------------------------------------------------------------------------------|
| `subject`    | Email subject line                                                                        |
| `sender`     | Sender display name shown by mail clients                                                 |
| *(any key)*  | Custom fields — accessible in the HTML layout template via `{{.frontmatter.yourField}}`  |

---

## Examples

### Confirm mail

```markdown
---
subject: "Please confirm your subscription"
sender: "My Newsletter"
---

Hi {{.Recipient.Name}},

Click below to confirm your subscription:

[Confirm my subscription]({{.confirmURL}})
```

### Newsletter body

```markdown
---
subject: "Issue #42"
sender: "My Newsletter"
---

Hello {{.Recipient.Name}},

Welcome to this week's edition…

[Unsubscribe]({{.unsubscribeURL}})
```

### HTML layout (`template.html`)

```html
<html>
  <head>
    <title>{{.metadata.Subject}}</title>
    <!-- custom frontmatter field -->
    <meta name="description" content="{{.frontmatter.description}}">
  </head>
  <body>
    {{.html}}
  </body>
</html>
```
