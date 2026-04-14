# 5000mails

A markdown-oriented newsletter platform. Signups, double opt-in, test-mails, markdown-based newsletters.

It comes with a backend to manage self-serving user setup (signup, opt-in, unsubscribe) and a CLI that uses a private management API to facilitate all interactions your minimal newsletter really needs.

This is a sibling project of [5000blogs](https://github.com/5000K/5000blogs).

## CLI

`5kmcli` is the command-line client for the management API. Build it with:

```sh
go build -o 5kmcli ./cmd/cli
```

**Global flags**

| Flag                 | Default                 | Description                             |
| -------------------- | ----------------------- | --------------------------------------- |
| `--server URL`       | `http://localhost:9000` | Server base URL                         |
| `--private-key-path` | -                       | Path to Ed25519 private key for signing |

**Commands**

```sh
# Mailing lists
5kmcli list create  --name NAME
5kmcli list get     --id ID
5kmcli list rename  --id ID --name NAME
5kmcli list delete  --id ID
5kmcli list users   --id ID

# Send newsletters
5kmcli send list  --list NAME  --raw-path PATH [--data KEY=VALUE ...]
5kmcli send test  --email EMAIL --raw-path PATH [--name NAME] [--data KEY=VALUE ...]

# Key management
5kmcli keys generate [--out-dir DIR]
```

`--data` can be repeated to inject multiple template variables, e.g. `--data title=Hello --data month=April`.

**Authentication**

Generate a key pair and configure the public key on the server:

```sh
5kmcli keys generate --out-dir ~/.config/5kmcli
```

Pass `--private-key-path ~/.config/5kmcli/5kmcli.key` on every subsequent call to sign requests automatically.
