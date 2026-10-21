# Meeting Assistant — Server

Go API for uploading meeting recordings, transcribing them, identifying
speakers and summarizing the conversation. The SvelteKit client is a static SPA
that authenticates against this service using Microsoft SSO and an httpOnly JWT
session cookie.

## Requirements

- Go 1.26+
- PostgreSQL with the `pgvector` extension (see `docker-compose.yaml`)
- `ffmpeg` and `ffprobe` on `PATH` (used to normalise uploads)
- `goose` and `sqlc` for migrations and query generation
- An Entra ID (Azure AD) app registration — see below

## Configuration

Copy these into `server/.env`:

| Variable | Description |
| --- | --- |
| `DATABASE_URL` | PostgreSQL connection string |
| `AWS_BUCKET_NAME` | Object storage bucket for recordings |
| `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_ENDPOINT_URL` | Storage credentials |
| `OPENROUTER_API_KEY` | Used for transcription and summaries |
| `EMBEDDING_SERVICE_URL` | URL of the Python embedding service |
| `AZURE_CLIENT_ID` | Entra app registration client ID |
| `AZURE_TENANT_ID` | Entra tenant (directory) ID |
| `AZURE_CLIENT_SECRET` | Entra app registration client secret |
| `AUTH_SECRET` | Random string used to sign session JWTs (`openssl rand -base64 48`) |
| `APP_URL` | Origin of the frontend, e.g. `http://localhost:5173` |
| `OAUTH_CALLBACK_URL` | Optional. Defaults to `http://localhost:8080/auth/callback` |

`Secure` cookies are enabled automatically when `APP_URL` starts with `https://`.

## Microsoft Entra ID app registration

1. Go to the [Entra admin center](https://entra.microsoft.com) → **Applications**
   → **App registrations** → **New registration**.
2. **Name**: `Meeting Assistant` (or anything you like).
3. **Supported account types**: *Accounts in this organizational directory only*
   — this restricts sign-in to your tenant.
4. **Redirect URI**: platform **Web**, value
   `http://localhost:8080/auth/callback` for local development. Add your
   production URL (for example `https://api.example.com/auth/callback`) as a
   second Web redirect URI when deploying.
5. Click **Register**, then copy:
   - **Application (client) ID** → `AZURE_CLIENT_ID`
   - **Directory (tenant) ID** → `AZURE_TENANT_ID`
6. Open **Certificates & secrets** → **New client secret**. Copy the secret
   **Value** immediately (it is only shown once) → `AZURE_CLIENT_SECRET`.
7. Open **Authentication** and make sure **Allow public client flows** is
   *disabled* (the backend performs a confidential-client code exchange) and the
   **ID tokens** checkbox is not required for the web flow used here.
8. No API permissions need to be added: the app uses the delegated
   `openid profile email` scopes from the Microsoft identity platform.

### How the flow works

```
Browser  → GET /auth/login?provider=microsoft
Backend  → 302 to login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize
           (authorization code + PKCE, state stored in a short-lived cookie)
Microsoft→ GET /auth/callback?code=…&state=…
Backend  → exchanges code, reads the ID token's `oid` claim,
           resolves/creates the user + account row,
           sets an httpOnly `meeting_session` cookie (HS256 JWT, 7 days),
           redirects back to APP_URL
```

Every endpoint except `/auth/login` and `/auth/callback` requires a valid
session. The session token is never exposed to JavaScript.

## Data model (auth)

- `app_user` — local identity: email, name, avatar.
- `account` — one row per linked external provider
  (`provider = 'microsoft'`, `provider_account_id = oid`). Adding another SSO
  provider means adding one provider implementation and new `account` rows; the
  `user` table does not change.
- `recording.user_id` — recordings belong to the uploading user.
- `speaker.user_id` — links a speaker (voice) to an application user. Voice
  matching is scoped per user, so embeddings never leak across accounts.
- `company.user_id` — companies are personal to their owner.

> **Note:** migration `0004` is destructive. It removes pre-auth data because
> those rows cannot be attributed to a real account.

## Development

```sh
make migrate-up     # apply migrations
make generate       # regenerate sqlc queries
go generate ./...   # regenerate the API from internal/api/api.yaml
make run            # run with air (hot reload)
```
