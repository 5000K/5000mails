# Configuration Reference

Configuration is loaded in two passes:

1. Environment variables are read first.
2. The YAML file at `CONFIG_PATH` (default: `config.yml`) is then parsed and merged on top.

YAML takes precedence over environment variables for every field that has both a `yaml:` tag and an `env:` tag. The config file path itself can only be set via `CONFIG_PATH`. If the file is not found, the server starts with environment-variable values only.

Path values (template, confirm-mail) accept either a local filesystem path or an `http(s)://` URL; the server fetches remote resources at startup.

---

## Top-level

| YAML key       | Environment variable | Default                   | Description                                                  |
|----------------|----------------------|---------------------------|--------------------------------------------------------------|
| `public-addr`  | `PUBLIC_ADDR`        | `:8080`                   | Listen address for the public API (subscribe/confirm/unsub)  |
| `private-addr` | `PRIVATE_ADDR`       | `:9000`                   | Listen address for the management API                        |
| `base-url`     | `BASE_URL`           | `http://localhost:8080`   | Public base URL; used to build confirmation links in emails  |

---

## `smtp`

| YAML key       | Environment variable  | Default              | Description                                              |
|----------------|-----------------------|----------------------|----------------------------------------------------------|
| `host`         | `SMTP_HOST`           | -                    | SMTP server hostname                                     |
| `port`         | `SMTP_PORT`           | `587`                | SMTP server port                                         |
| `username`     | `SMTP_USERNAME`       | -                    | SMTP authentication username                             |
| `password`     | `SMTP_PASSWORD`       | -                    | SMTP authentication password                             |
| `sender-email` | `SMTP_SENDER_EMAIL`   | -                    | From address used for all outgoing mail                  |
| `tls-policy`   | `SMTP_TLS_POLICY`     | `TLSOpportunistic`   | One of `TLSMandatory`, `TLSOpportunistic`, or `NoTLS`   |

---

## `db`

| YAML key | Environment variable | Default         | Description                                                  |
|----------|----------------------|-----------------|--------------------------------------------------------------|
| `type`   | `DB_TYPE`            | `sqlite`        | Database driver: `sqlite` or `postgres`                      |
| `dsn`    | `DB_DSN`             | `5000mails.db`  | SQLite file path, or a Postgres connection string            |

**Postgres DSN example**

```
host=db port=5432 user=mails password=secret dbname=mails sslmode=disable
```

---

## `auth`

| YAML key           | Environment variable   | Default | Description                                                                               |
|--------------------|------------------------|---------|-------------------------------------------------------------------------------------------|
| `public-key-path`  | `AUTH_PUBLIC_KEY_PATH` | -       | Path to an Ed25519 public key. Leave empty to disable request signing on the management API. |

Generate a key pair with the CLI:

```sh
5kmcli keys generate --out-dir ~/.config/5kmcli
# writes 5kmcli.key (private) and 5kmcli.pub (public)
```

Set `auth.public-key-path` to the `.pub` file and pass `--private-key-path` to every CLI invocation.

---

## `paths`

All values accept a local path **or** an `http(s)://` URL. Remote resources are fetched once at startup.

| YAML key       | Environment variable   | Default (remote)                                                                      | Description                                                 |
|----------------|------------------------|---------------------------------------------------------------------------------------|-------------------------------------------------------------|
| `template`     | `TEMPLATE_PATH`        | `https://github.com/5000K/5000mails/releases/latest/download/template.html`           | HTML wrapper rendered around every markdown newsletter      |
| `confirm-mail` | `CONFIRM_MAIL_PATH`    | `https://github.com/5000K/5000mails/releases/latest/download/confirm.md`              | Markdown template for the double opt-in confirmation email  |

See [docs/TEMPLATE.md](TEMPLATE.md) for a full reference of template variables available in the `confirm-mail` template and all other mail contexts.

---

## `redirects`

When set, the public API issues a `303 See Other` redirect to the configured URL instead of returning a JSON response. Leave any field blank to keep the default JSON behaviour for that outcome.

| YAML key              | Environment variable              | Default | Description                                            |
|-----------------------|-----------------------------------|---------|--------------------------------------------------------|
| `subscribe-success`   | `REDIRECT_SUBSCRIBE_SUCCESS`      | -       | Redirect after a successful subscription request       |
| `subscribe-error`     | `REDIRECT_SUBSCRIBE_ERROR`        | -       | Redirect on a bad subscription request or server error |
| `confirm-success`     | `REDIRECT_CONFIRM_SUCCESS`        | -       | Redirect after a token is confirmed                    |
| `confirm-error`       | `REDIRECT_CONFIRM_ERROR`          | -       | Redirect on an invalid or expired confirmation token   |
| `unsubscribe-success` | `REDIRECT_UNSUBSCRIBE_SUCCESS`    | -       | Redirect after a successful unsubscribe                |
| `unsubscribe-error`   | `REDIRECT_UNSUBSCRIBE_ERROR`      | -       | Redirect on an invalid or expired unsubscribe token    |

---

## Full example

```yaml
public-addr: ":8080"
private-addr: ":9000"
base-url: "https://newsletter.yoursite.com"

smtp:
  host: "smtp.example.com"
  port: 587
  username: "you@example.com"
  password: "secret"
  sender-email: "newsletter@yoursite.com"
  tls-policy: "TLSOpportunistic"

db:
  type: "sqlite"
  dsn: "5000mails.db"

auth:
  public-key-path: "/etc/5000mails/5kmcli.pub"

paths:
  template: "./static/template.html"
  confirm-mail: "./static/confirm.md"

redirects:
  subscribe-success: "https://yoursite.com/subscribed"
  subscribe-error: "https://yoursite.com/subscribe-failed"
  confirm-success: "https://yoursite.com/confirmed"
  confirm-error: "https://yoursite.com/confirm-failed"
  unsubscribe-success: "https://yoursite.com/unsubscribed"
  unsubscribe-error: "https://yoursite.com/unsubscribe-failed"
```
