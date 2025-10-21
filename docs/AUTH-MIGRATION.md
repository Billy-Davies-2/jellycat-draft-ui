# Auth migration: NextAuth -> Better Auth

This app now uses Better Auth with the Generic OAuth plugin, configured for Authentik via environment variables.

## Routes
- Catch-all handler: `/api/auth/[...all]` handled by Better Auth.
- OAuth2 callback is automatically mounted at `/api/auth/oauth2/callback/:providerId` (we use `authentik`).

## Middleware
`/admin` is protected by checking Better Auth session cookie. Unauthenticated users are redirected to the Better Auth generic OAuth sign-in endpoint preselected for Authentik.

## Required environment variables
Set these in your Helm chart or deployment environment:

- AUTH_URL: Public base URL of your app (e.g. `https://draft.example.com`).
- BETTER_AUTH_SECRET: A long random string for signing cookies/tokens.
- AUTHENTIK_URL: Base URL of your Authentik issuer (e.g. `https://auth.example.com`).
- AUTHENTIK_CLIENT_ID: OIDC client ID from Authentik application.
- AUTHENTIK_CLIENT_SECRET: OIDC client secret from Authentik application.

### Database
- production: set `DATABASE_URL` (or `POSTGRES_URL`) for Postgres, e.g. `postgres://user:pass@host:5432/dbname`.
- development: optional `AUTH_SQLITE_PATH` for local sqlite file (defaults to `.auth.sqlite`).

Optional fallbacks handled by the code:
- NEXTAUTH_URL, NEXT_PUBLIC_APP_URL, or PUBLIC_BASE_URL can be used in place of AUTH_URL.
- AUTHENTIK_ISSUER can be used instead of AUTHENTIK_URL.

## Authentik application settings
- Redirect URI: `https://draft.example.com/api/auth/oauth2/callback/authentik`
- Scopes: `openid profile email groups`

## Notes
- By default, Better Auth uses in-memory storage when no database is configured. For production, add a DB adapter (e.g. Drizzle/Prisma/SQLite/Postgres). See Better Auth docs.
- Client-side helpers (e.g. `authClient.signIn.oauth2`) are not wired yet; the middleware uses server endpoints for redirects.
