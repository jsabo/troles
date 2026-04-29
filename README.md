# troles

Show a Teleport user's effective roles, including any roles granted by access lists — the information missing from the Teleport web UI's Users page.

## Install

```sh
brew install jsabo/tap/troles
```

Or build from source:

```sh
go install github.com/jsabo/troles/cmd/troles@latest
```

## Usage

```
troles [flags] [username]
```

If `username` is omitted, the currently logged-in tsh user is used.

```sh
# Current user
troles

# Specific user
troles alice@example.com

# JSON output for scripting
troles --format json alice@example.com

# Explicit proxy / cluster
troles --proxy teleport.example.com:443 alice@example.com
```

### Example output

```
User:  alice@example.com

Base roles:
  access
  editor

Access list grants:
  db-readonly
  node-admin

Effective roles:
  access, db-readonly, editor, node-admin
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--proxy` | active tsh profile | Teleport proxy address |
| `--cluster` | active tsh profile | tsh profile name (proxy host) to use |
| `--tsh-profile-dir` | `~/.tsh` | tsh profile directory |
| `--format` | `table` | Output format: `table` or `json` |
| `--verbose` | — | Print full connection error detail |
| `--version` | — | Print version and exit |

## tsh alias

Wire `troles` up as `tsh roles` by adding to `~/.tsh/config/config.yaml`:

```yaml
aliases:
  roles: /usr/local/bin/troles
```

Then:

```sh
tsh roles alice@example.com
tsh roles  # current user
```

## Requirements

- An active `tsh login` session
- Permission to read `user_login_state` resources (admin role or equivalent)

## License

Apache 2.0
