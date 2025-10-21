import { betterAuth } from "better-auth";
import { genericOAuth } from "better-auth/plugins";

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

export const auth = betterAuth({
  baseURL: `${BASE_URL}/api/auth`,
  // session cookies config can be customized here if needed
  plugins: [
    genericOAuth({
      config: [
        {
          providerId: "authentik",
          clientId: AUTHENTIK_CLIENT_ID,
          clientSecret: AUTHENTIK_CLIENT_SECRET,
          // Use OIDC discovery if base URL provided
          discoveryUrl: AUTHENTIK_URL
            ? `${AUTHENTIK_URL.replace(/\/$/, "")}/.well-known/openid-configuration`
            : undefined,
          scopes: ["openid", "profile", "email", "groups"],
          // Ensure redirect URI aligns with Better Auth default callback
          // <BASE_URL>/api/auth/oauth2/callback/<providerId>
          redirectURI: `${BASE_URL.replace(/\/$/, "")}/api/auth/oauth2/callback/authentik`,
          // Minimal mapping to ensure user objects have id/email/name
          mapProfileToUser: async (profile: any) => {
            // Try common OIDC fields; fall back gracefully
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
