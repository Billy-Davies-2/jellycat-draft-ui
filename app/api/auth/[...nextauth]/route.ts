import NextAuth from "next-auth";
import { OAuthConfig } from "next-auth/providers/oauth";

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
    } as OAuthConfig<any>,
  ],
  // Rely on NEXTAUTH_URL provided by the environment
  trustHost: true,
  // Recommended for stateless deployments
  session: { strategy: "jwt" as const },
};

const handler = NextAuth(authOptions);
export { handler as GET, handler as POST };
