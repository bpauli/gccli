# State that gccli stores

This page lists every file and keyring item that gccli writes, the format of each, and the rules that gccli uses to read and write them. It describes gccli as built from this repository.

## Config directory

gccli keeps its files in the `gccli` directory under the user config directory that Go's `os.UserConfigDir` returns.

| OS | Config directory |
| --- | --- |
| Linux and other Unix | `$XDG_CONFIG_HOME/gccli`, or `~/.config/gccli` when `XDG_CONFIG_HOME` is not set |
| macOS | `~/Library/Application Support/gccli` |
| Windows | `%AppData%\gccli` |

The directory holds these entries. gccli creates the directory with mode `0700` when it writes the first file.

| Entry | Content |
| --- | --- |
| `config.json` | Settings. See [config.json](#configjson). |
| `config.lock` | Lock for writes to `config.json`. See [Lock files](#lock-files). |
| `token.lock` | Lock for token refresh. See [Lock files](#lock-files). |
| `credentials/` | Token files of the `file` keyring backend. See [File backend](#file-backend). |

## config.json

`config.json` is one JSON object. Every key is optional. gccli reads a missing file as an empty object.

| Key | Type | Meaning |
| --- | --- | --- |
| `default_account` | string | Account email that gccli uses when you do not give `--account` or `GCCLI_ACCOUNT`. `auth login` and `auth import` set it. |
| `keyring_backend` | string | Keyring backend. The values are `keychain`, `secret-service` and `file`. Any other value, or no value, selects the backend automatically. |
| `domain` | string | Garmin domain, `garmin.com` or `garmin.cn`. See [Domain](#domain). |
| `default_format` | string | gccli parses this key, but no command reads it. |
| `activity_summary` | object | Maps an activity category to a list of field names. `activity summary` shows these fields for that category. |
| `units` | string | `metric`, `statute_us`, `statute_uk` or `auto`, in any letter case. An invalid value gives a warning on stderr, and gccli then uses `auto`. |

gccli keeps keys that it does not know. It never removes them.

### Write rule

`config.Update` is the only code that writes `config.json`. It does these steps:

1. It creates the config directory with mode `0700` if the directory does not exist.
2. It takes an exclusive lock on `config.lock`.
3. If `config.json` is a symbolic link, it resolves the link and writes to the target file.
4. It reads the file. A missing file reads as `{}`. If the file is not valid JSON, `Update` returns an error and does not write.
5. It decodes the known keys, applies the change, and writes every known key again from the result. A known key that is empty after the change is removed. Unknown keys keep their values.
6. It writes the result, indented with two spaces and followed by a newline, to a temporary file in the same directory. It syncs that file and renames it over `config.json`. The new file has mode `0600`.

## Lock files

Both lock files are in the config directory. gccli creates each one with mode `0600` and never deletes it. The file content is empty. The lock is an exclusive `flock` on the open file, and the operating system releases it when the process exits.

| File | Held while |
| --- | --- |
| `config.lock` | `config.Update` reads, changes and writes `config.json`. |
| `token.lock` | gccli reads the stored token again, refreshes it with Garmin, and stores the result. A second process that waits on the lock then reads the new token. It does not refresh a second time. |

When a process waits for `token.lock` for more than one second, it writes `Waiting for another gccli process to refresh the token` to stderr.

On systems that are not Unix, such as Windows, `LockFile` does nothing. Writes to `config.json` are atomic but not locked there, and token refreshes are not locked.

## Keyring

gccli stores one token per account in the keyring, through `github.com/99designs/keyring`.

| Property | Value |
| --- | --- |
| Service name | `gccli` |
| Item key | `gccli:token:<email>` |
| Item data | The token JSON. See [Token](#token). |

### Backend

gccli picks the backend in this order. The first one that is set wins.

1. `GCCLI_KEYRING_BACKEND`
2. `keyring_backend` in `config.json`
3. Automatic selection by the keyring library

On Linux, when neither `GCCLI_KEYRING_BACKEND` nor `keyring_backend` has a value and automatic selection cannot open a keyring, gccli uses the `file` backend. A value that selects the backend automatically, such as `auto`, gets no `file` fallback. On macOS, the Keychain item is not synchronized, it is readable only when the Keychain is unlocked, and it trusts the gccli binary.

### File backend

The `file` backend writes one file per item in `credentials/` in the config directory. The file name is the item key, for example `gccli:token:you@example.com`. The file has mode `0600`. The directory has mode `0700`.

gccli writes the new file in a temporary directory in `credentials/` and then renames it over the old file. A process that reads the file at the same time gets the old or the new token, never a partly written file. A temporary directory that a crash leaves behind has a name that starts with `.set-`. You can delete it.

Each file is a JWE (PBES2-HS256+A128KW key wrap, A256GCM content encryption) of the keyring item. The passphrase is the fixed string `gccli-file-store`, built into gccli. You cannot change it. Anyone who can read the file can decrypt the token.

## Token

The token is one JSON object. `auth export` prints it as standard base64. `auth import` reads that base64 form.

| Key | Type | Meaning |
| --- | --- | --- |
| `version` | number | Format version. gccli writes `1`. See [Version rule](#version-rule). |
| `oauth1_token` | string | OAuth1 token. `auth login` leaves it empty. When a token has OAuth1 credentials and no `di_client_id`, gccli refreshes with OAuth1. |
| `oauth1_secret` | string | OAuth1 token secret. `auth login` leaves it empty. |
| `oauth2_access_token` | string | Bearer token that gccli sends to the Garmin Connect API. |
| `oauth2_refresh_token` | string | DI refresh token. |
| `oauth2_expires_at` | string | Expiry time of the access token, RFC 3339. gccli treats the token as expired 60 seconds before this time. The zero time counts as expired. |
| `di_client_id` | string | DI OAuth client id that issued the refresh token. gccli omits the key when it is empty. |
| `mfa_token` | string | MFA token from the login. gccli omits the key when it is empty. |
| `domain` | string | `garmin.com`, `garmin.cn`, or empty. See [Domain](#domain). |
| `email` | string | Account email. `auth import` fails when it is empty. |

gccli ignores keys that it does not know. gccli 2.0.0 and older declared a `display_name` key but never wrote it. gccli ignores that key too, and the next write of the token drops it.

### Version rule

- A token without `version` is version 1.
- gccli reads version 1.
- A `version` of `null` is version 1.
- For any other integer version, gccli fails before it sends a request. The error names the version of the token and version 1. A `version` that is not an integer fails with the JSON decoder error. `auth import` stores nothing in either case.
- `auth login`, `auth import` and every token refresh write the token with `"version":1`. They write only the keys in the table above.

gccli 2.0.0 ignores the `version` key, so it reads a version 1 token.

gccli drops unknown keys each time it writes a token. A key that must survive a write by an older gccli therefore needs a new version.

## Domain

`auth login` uses the first domain that is set:

1. `GCCLI_DOMAIN`
2. `domain` in `config.json`
3. `garmin.com`

The login writes that domain into the token. API calls and token refresh use the `domain` of the token. When the token has an empty `domain`, they use the same order as `auth login`. gccli accepts only `garmin.com` and `garmin.cn`. Any other value fails with an error before gccli sends a request.

When the `domain` of the stored token has another value, the error names the stored token and tells you to run `gccli auth login <email>`. `auth status` and `auth token` check the domain in the same way as API calls. They fail with the same errors and exit code 1.

## Account

gccli uses the first account that is set:

1. `--account`, or `GCCLI_ACCOUNT` when the flag is not given
2. `default_account` in `config.json`
