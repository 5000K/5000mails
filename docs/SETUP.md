# Setup

## Prerequisites

You need an SMTP server to send mail. All other dependencies (database, templates) are handled by the server itself.

---

## Docker

```sh
docker run -d \
  --name 5000mails \
  -p 8080:8080 \
  -p 9000:9000 \
  -v $(pwd)/config.yml:/config.yml \
  -v $(pwd)/data:/data \
  -e CONFIG_PATH=/config.yml \
  ghcr.io/5000k/5000mails:latest
```

Mount a directory for the SQLite database file or point `db.dsn` at a Postgres instance.

---

## Docker Compose

```yaml
services:
  5000mails:
    image: ghcr.io/5000k/5000mails:latest
    restart: unless-stopped
    ports:
      - "8080:8080"   # public API
      - "9000:9000"   # management API (keep this private)
    volumes:
      - ./config.yml:/config.yml
      - ./data:/data
    environment:
      CONFIG_PATH: /config.yml
```

Create a `config.yml` next to the compose file (see [CONFIG.md](CONFIG.md) for all options). A minimal working example:

```yaml
base-url: "https://newsletter.yoursite.com"

smtp:
  host: "smtp.example.com"
  port: 587
  username: "you@example.com"
  password: "secret"
  sender-email: "newsletter@yoursite.com"

db:
  type: "sqlite"
  dsn: "/data/5000mails.db"
```

Then start:

```sh
docker compose up -d
```

---

## Binary

Prebuilt binaries are available for every release at:

**<https://github.com/5000K/5000mails/releases/latest>**

| Platform        | Architecture | File                                    |
|-----------------|--------------|-----------------------------------------|
| Linux           | x86\_64      | `5000mails-linux-amd64`                 |
| Linux           | AArch64      | `5000mails-linux-arm64`                 |
| Windows         | x86\_64      | `5000mails-windows-amd64.exe`           |
| Windows         | AArch64      | `5000mails-windows-arm64.exe`           |

The `5kmcli` management client is published alongside the server binary.

### Linux quickstart

```sh
# Download the server binary
curl -Lo 5000mails https://github.com/5000K/5000mails/releases/latest/download/5000mails-linux-amd64
chmod +x 5000mails

# Download the CLI
curl -Lo 5kmcli https://github.com/5000K/5000mails/releases/latest/download/5kmcli-linux-amd64
chmod +x 5kmcli

# Create a minimal config
cat > config.yml <<'EOF'
base-url: "http://localhost:8080"

smtp:
  host: "smtp.example.com"
  port: 587
  username: "you@example.com"
  password: "secret"
  sender-email: "newsletter@yoursite.com"
EOF

# Run
CONFIG_PATH=config.yml ./5000mails
```

### Running as a systemd service

```ini
[Unit]
Description=5000mails newsletter server
After=network.target

[Service]
ExecStart=/usr/local/bin/5000mails
Environment=CONFIG_PATH=/etc/5000mails/config.yml
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

```sh
sudo cp 5000mails /usr/local/bin/
sudo mkdir -p /etc/5000mails
sudo cp config.yml /etc/5000mails/
sudo systemctl daemon-reload
sudo systemctl enable --now 5000mails
```

---

## Authentication setup

The management API supports optional Ed25519 request signing. Generate a key pair with the CLI and configure the public key on the server.

```sh
./5kmcli keys generate --out-dir ~/.config/5kmcli
# Writes:
#   ~/.config/5kmcli/5kmcli.key  (private - keep this secret)
#   ~/.config/5kmcli/5kmcli.pub  (public - put this on the server)
```

Add to `config.yml`:

```yaml
auth:
  public-key-path: "/etc/5000mails/5kmcli.pub"
```

All subsequent CLI invocations need `--private-key-path`:

```sh
./5kmcli --private-key-path ~/.config/5kmcli/5kmcli.key list all
```
