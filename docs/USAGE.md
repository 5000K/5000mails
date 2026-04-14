# Usage

All management operations go through `5kmcli`, the command-line client for the management API.

**Global flags** (available on every command):

| Flag                    | Default                   | Description                              |
|-------------------------|---------------------------|------------------------------------------|
| `--server URL`          | `http://localhost:9000`   | Management API base URL                  |
| `--private-key-path`    | -                         | Path to Ed25519 private key for signing  |

For brevity the examples below omit `--server` and `--private-key-path`. Add them as needed:

```sh
5kmcli --server https://mails.yoursite.com \
       --private-key-path ~/.config/5kmcli/5kmcli.key \
       list all
```

---

## Managing mailing lists

### Create a list

```sh
5kmcli list create --name weekly
```

### List all lists

```sh
5kmcli list all
```

### Get details and subscriber counts

```sh
5kmcli list get --name weekly
```

### Rename a list

```sh
5kmcli list rename --name weekly --new-name monthly
```

### Delete a list

```sh
5kmcli list delete --name weekly
```

### View subscribers

```sh
5kmcli list users --name weekly
```

---

## Authoring a newsletter

Newsletters are written in Markdown. Create a file, e.g. `issue-42.md`:

```markdown
# Issue 42 - {{.title}}

Hello {{.name}},

This is the latest edition of the newsletter.

[Read more on the blog](https://yoursite.com/posts/42)
```

Template variables are injected with `--data KEY=VALUE` at send time. Any key from `--data` is available in the template as `{{.KEY}}`.

---

## Sending a test mail

Always send a test before dispatching to the full list.

```sh
5kmcli send test \
  --email   you@yoursite.com \
  --name    "Your Name" \
  --raw-path issue-42.md \
  --data    title="April Update" \
  --data    name="Friend"
```

The test mail is rendered and delivered immediately to the single recipient without touching any list.

---

## Sending to a list

Once the test looks good, send to all confirmed subscribers:

```sh
5kmcli send list \
  --list    weekly \
  --raw-path issue-42.md \
  --data    title="April Update" \
  --data    name="Friend"
```

Only subscribers who completed the double opt-in confirmation are included.

---

## Subscribe form (HTML)

The public API accepts both JSON and standard HTML form submissions, so you can embed a plain `<form>` on any page with no JavaScript required.

Replace `https://newsletter.yoursite.com` with your actual `base-url` and `weekly` with your list name.

```html
<form
  action="https://newsletter.yoursite.com/weekly/subscribe"
  method="POST"
>
  <label for="name">Name</label>
  <input id="name" name="name" type="text" required placeholder="Your name" />

  <label for="email">Email</label>
  <input id="email" name="email" type="email" required placeholder="you@example.com" />

  <button type="submit">Subscribe</button>
</form>
```

After submitting:

- **Without redirects configured** - the server returns a JSON `202 Accepted` response. You should display a success message with JavaScript or redirect the user yourself.
- **With redirects configured** - the server issues a `303 See Other` to the URL you set in `redirects.subscribe-success` (or `redirects.subscribe-error` on failure), so the browser lands on your custom page automatically.

See [CONFIG.md](CONFIG.md#redirects) for redirect configuration.

The confirmation email is sent automatically. The subscriber clicks the link in that email, which hits `GET /confirm/{token}`, and their subscription becomes active.

Every newsletter automatically includes an unsubscribe link that hits `GET /unsubscribe/{token}`.
