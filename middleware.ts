import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { getSessionCookie } from "better-auth/cookies";

// Protect admin route: only allow users in Authentik group "homelab-admins".
export async function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  if (!pathname.startsWith("/admin")) {
    return NextResponse.next();
  }

  // If there is a Better Auth session cookie, allow access.
  const session = getSessionCookie(req);
  if (session) {
    // TODO: If you need admin flag, consider fetching session from server in a protected API or encoding roles in session custom claims.
    return NextResponse.next();
  }

  // Not authenticated -> redirect to Better Auth generic OAuth sign-in (Authentik)
  const base = process.env.AUTH_URL || process.env.NEXTAUTH_URL || process.env.NEXT_PUBLIC_APP_URL || process.env.PUBLIC_BASE_URL;
  const baseUrl = base ? new URL(base) : new URL(req.url);
  // Better Auth oauth2 sign-in endpoint
  const signInUrl = new URL("/api/auth/sign-in/oauth2", baseUrl);
  const callbackUrl = new URL(req.nextUrl.pathname + req.nextUrl.search, baseUrl);
  signInUrl.searchParams.set("providerId", "authentik");
  signInUrl.searchParams.set("callbackURL", callbackUrl.toString());
  return NextResponse.redirect(signInUrl);
}

export const config = {
  matcher: ["/admin/:path*"],
};
