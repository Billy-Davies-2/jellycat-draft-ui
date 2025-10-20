import { OAuthConfig } from "next-auth/providers/oauth";

export const authCallbacks = {
  async jwt({ token, account, profile }: any) {
    const fromProfile = (profile?.groups ?? profile?.group ?? profile?.roles) as
      | string[]
      | string
      | undefined;
    if (fromProfile) {
      token.groups = Array.isArray(fromProfile) ? fromProfile : [fromProfile];
    }
    const idt = (account as any)?.id_token as string | undefined;
    let idClaims: any = undefined;
    if (idt) {
      try {
        const body = JSON.parse(Buffer.from(idt.split(".")[1], "base64url").toString());
        idClaims = body;
        const g = body?.groups ?? body?.group ?? body?.roles;
        if (g) token.groups = Array.isArray(g) ? g : [g];
      } catch {}
    }

    // Compute admin flag from a configurable claim/value
    const claim = process.env.AUTH_ADMIN_CLAIM;
    const expected = process.env.AUTH_ADMIN_VALUE;
    if (claim) {
      const sourceVal = (profile && (profile as any)[claim]) ?? (idClaims && idClaims[claim]);
      if (sourceVal !== undefined) {
        if (expected && expected !== "") {
          if (Array.isArray(sourceVal)) token.isAdmin = sourceVal.includes(expected);
          else token.isAdmin = String(sourceVal) === expected;
        } else {
          if (typeof sourceVal === "boolean") token.isAdmin = sourceVal;
          else if (Array.isArray(sourceVal)) token.isAdmin = sourceVal.length > 0;
          else token.isAdmin = Boolean(sourceVal);
        }
      }
    }
    return token;
  },
  async session({ session, token }: any) {
    (session as any).groups = (token as any).groups ?? [];
    (session as any).isAdmin = (token as any).isAdmin === true;
    return session;
  },
};

export const authOptions = {
  providers: [
    {
      id: "authentik",
      name: "Authentik",
      type: "oauth",
      issuer: process.env.AUTHENTIK_ISSUER,
      clientId: process.env.AUTHENTIK_CLIENT_ID,
      clientSecret: process.env.AUTHENTIK_CLIENT_SECRET,
      wellKnown: `${process.env.AUTHENTIK_URL}/.well-known/openid-configuration`,
      authorization: {
        params: {
          scope: "openid profile email groups",
        },
      },
    } as OAuthConfig<any>,
  ],
  trustHost: true,
  session: { strategy: "jwt" as const },
  callbacks: authCallbacks,
};
