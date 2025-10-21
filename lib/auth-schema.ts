import { Kysely, sql } from "kysely";

// Creates minimal Better Auth tables if they do not exist.
// Idempotent: safe to run on startup.
export async function ensureAuthSchemaPostgres(db: Kysely<any>) {
  // users (lowercase table, camelCase columns used by Better Auth)
  await sql`
    CREATE TABLE IF NOT EXISTS users (
      id TEXT PRIMARY KEY,
      email TEXT NOT NULL UNIQUE,
      name TEXT,
      image TEXT,
      "emailVerified" BOOLEAN NOT NULL DEFAULT FALSE,
      "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )
  `.execute(db);
  // Backfill missing camelCase columns if table existed with snake_case
  await sql`ALTER TABLE users ADD COLUMN IF NOT EXISTS "emailVerified" BOOLEAN NOT NULL DEFAULT FALSE`.execute(db);
  await sql`ALTER TABLE users ADD COLUMN IF NOT EXISTS "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()`.execute(db);
  await sql`ALTER TABLE users ADD COLUMN IF NOT EXISTS "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()`.execute(db);

  // account (lowercase table, camelCase columns)
  await sql`
    CREATE TABLE IF NOT EXISTS account (
      id BIGSERIAL PRIMARY KEY,
      "userId" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      "providerId" TEXT NOT NULL,
      "accountId" TEXT NOT NULL,
      "accessToken" TEXT,
      "refreshToken" TEXT,
      "idToken" TEXT,
      "accessTokenExpiresAt" TIMESTAMPTZ,
      "refreshTokenExpiresAt" TIMESTAMPTZ,
      scope TEXT,
      password TEXT,
      "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      CONSTRAINT account_provider_account_unique UNIQUE ("providerId", "accountId")
    )
  `.execute(db);
  await sql`ALTER TABLE account ADD COLUMN IF NOT EXISTS "userId" TEXT`.execute(db);
  await sql`ALTER TABLE account ADD COLUMN IF NOT EXISTS "providerId" TEXT`.execute(db);
  await sql`ALTER TABLE account ADD COLUMN IF NOT EXISTS "accountId" TEXT`.execute(db);
  await sql`ALTER TABLE account ADD COLUMN IF NOT EXISTS "accessToken" TEXT`.execute(db);
  await sql`ALTER TABLE account ADD COLUMN IF NOT EXISTS "refreshToken" TEXT`.execute(db);
  await sql`ALTER TABLE account ADD COLUMN IF NOT EXISTS "idToken" TEXT`.execute(db);
  await sql`ALTER TABLE account ADD COLUMN IF NOT EXISTS "accessTokenExpiresAt" TIMESTAMPTZ`.execute(db);
  await sql`ALTER TABLE account ADD COLUMN IF NOT EXISTS "refreshTokenExpiresAt" TIMESTAMPTZ`.execute(db);
  await sql`ALTER TABLE account ADD COLUMN IF NOT EXISTS "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()`.execute(db);
  await sql`ALTER TABLE account ADD COLUMN IF NOT EXISTS "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()`.execute(db);

  // session (lowercase table, camelCase columns)
  await sql`
    CREATE TABLE IF NOT EXISTS session (
      token TEXT PRIMARY KEY,
      "userId" TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      "expiresAt" TIMESTAMPTZ NOT NULL,
      "ipAddress" TEXT,
      "userAgent" TEXT,
      "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )
  `.execute(db);
  await sql`ALTER TABLE session ADD COLUMN IF NOT EXISTS "userId" TEXT`.execute(db);
  await sql`ALTER TABLE session ADD COLUMN IF NOT EXISTS "expiresAt" TIMESTAMPTZ`.execute(db);
  await sql`ALTER TABLE session ADD COLUMN IF NOT EXISTS "ipAddress" TEXT`.execute(db);
  await sql`ALTER TABLE session ADD COLUMN IF NOT EXISTS "userAgent" TEXT`.execute(db);
  await sql`ALTER TABLE session ADD COLUMN IF NOT EXISTS "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()`.execute(db);
  await sql`ALTER TABLE session ADD COLUMN IF NOT EXISTS "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()`.execute(db);

  // verification (lowercase table, camelCase columns)
  await sql`
    CREATE TABLE IF NOT EXISTS verification (
      id BIGSERIAL PRIMARY KEY,
      identifier TEXT NOT NULL UNIQUE,
      value TEXT NOT NULL,
      "expiresAt" TIMESTAMPTZ NOT NULL,
      "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )
  `.execute(db);
  await sql`ALTER TABLE verification ADD COLUMN IF NOT EXISTS "expiresAt" TIMESTAMPTZ`.execute(db);
  await sql`ALTER TABLE verification ADD COLUMN IF NOT EXISTS "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()`.execute(db);
  await sql`ALTER TABLE verification ADD COLUMN IF NOT EXISTS "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()`.execute(db);

  // rate limit storage: table name is case-sensitive in queries (model: "rateLimit")
  await sql`
    CREATE TABLE IF NOT EXISTS "rateLimit" (
      key TEXT PRIMARY KEY,
      count INTEGER NOT NULL DEFAULT 0,
      "lastRequest" BIGINT NOT NULL DEFAULT 0
    )
  `.execute(db);
  // Also ensure commonly-created snake_case fallback exists (optional)
  await sql`
    CREATE TABLE IF NOT EXISTS ratelimit (
      key TEXT PRIMARY KEY,
      count INTEGER NOT NULL DEFAULT 0,
      last_request BIGINT NOT NULL DEFAULT 0
    )
  `.execute(db);
}
