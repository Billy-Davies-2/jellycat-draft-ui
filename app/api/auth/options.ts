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
    if (idt) {
      try {
        const body = JSON.parse(Buffer.from(idt.split(".")[1], "base64url").toString());
        const g = body?.groups ?? body?.group ?? body?.roles;
        if (g) token.groups = Array.isArray(g) ? g : [g];
      } catch {}
    }
    return token;
  },
  async session({ session, token }: any) {
    (session as any).groups = (token as any).groups ?? [];
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
