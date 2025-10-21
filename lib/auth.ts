import { betterAuth } from "better-auth";
import { genericOAuth } from "better-auth/plugins";
import Database from "better-sqlite3";
import { Kysely, PostgresDialect } from "kysely";
import { Pool } from "pg";
import { ensureAuthSchemaPostgres } from "@/lib/auth-schema";

// Base URL for Better Auth to construct callback routes
// Prefer AUTH_URL (generic), then NEXTAUTH_URL, then NEXT_PUBLIC_APP_URL, then PUBLIC_BASE_URL
const BASE_URL =
  process.env.AUTH_URL ||
  process.env.NEXTAUTH_URL ||
  process.env.NEXT_PUBLIC_APP_URL ||
  process.env.PUBLIC_BASE_URL ||
  "http://localhost:3000";

// Authentik OIDC configuration via env
const AUTHENTIK_URL = process.env.AUTHENTIK_URL || process.env.AUTHENTIK_ISSUER || "";
const AUTHENTIK_CLIENT_ID = process.env.AUTHENTIK_CLIENT_ID || "";
const AUTHENTIK_CLIENT_SECRET = process.env.AUTHENTIK_CLIENT_SECRET || "";

// In development without DB, Better Auth falls back to in-memory storage.
// For production, configure a database adapter per docs.

// Configure database adapter: Postgres in production, SQLite in development
function getDatabaseOption(): any {
  if (process.env.NODE_ENV === "production") {
    const connectionString = process.env.DATABASE_URL || process.env.POSTGRES_URL || "";
    const pool = new Pool({ connectionString });
    const kysely = new Kysely({ dialect: new PostgresDialect({ pool }) });
    // Ensure required tables exist (users, account, session, verification, ratelimit)
    // This runs once at startup in the app process; it is idempotent.
    ensureAuthSchemaPostgres(kysely).catch((err) => {
      console.error("Failed to ensure Better Auth schema:", err);
    });
    return {
      database: {
        db: kysely,
        type: "postgres" as const,
      },
    };
  }
  // dev: use a local SQLite file
  const sqlitePath = process.env.AUTH_SQLITE_PATH || "dev.sqlite";
  const db = new Database(sqlitePath);
  return {
    database: db,
  };
}

export const auth = betterAuth({
  ...getDatabaseOption(),
  baseURL: `${BASE_URL}/api/auth`,
  secret: process.env.BETTER_AUTH_SECRET,
  emailAndPassword: {
    enabled: true,
    autoSignIn: true,
  },
  plugins: [
    genericOAuth({
      config: [
        {
          providerId: "authentik",
          clientId: AUTHENTIK_CLIENT_ID,
          clientSecret: AUTHENTIK_CLIENT_SECRET,
          discoveryUrl: AUTHENTIK_URL
            ? `${AUTHENTIK_URL.replace(/\/$/, "")}/.well-known/openid-configuration`
            : undefined,
          scopes: ["openid", "profile", "email", "groups"],
          redirectURI: `${BASE_URL.replace(/\/$/, "")}/api/auth/oauth2/callback/authentik`,
          mapProfileToUser: async (profile: any) => {
            const id = profile?.sub || profile?.id || profile?.uid || "";
            const email = profile?.email || profile?.preferred_username || undefined;
            const name = profile?.name || profile?.display_name || profile?.preferred_username || undefined;
            const image = profile?.picture || undefined;
            return { id, email, name, image } as any;
          },
        },
      ],
    }),
  ],
});

export type Auth = typeof auth;
