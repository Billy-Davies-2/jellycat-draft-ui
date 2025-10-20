import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { getToken } from "next-auth/jwt";

// Protect admin route: only allow users in Authentik group "homelab-admins".
export async function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  if (!pathname.startsWith("/admin")) {
    return NextResponse.next();
  }

  const token = await getToken({ req, secret: process.env.NEXTAUTH_SECRET });
  const groups: string[] = (token as any)?.groups ?? [];

  if (groups.includes("homelab-admins")) {
    return NextResponse.next();
  }

  // Not authorized -> redirect to sign in
  const signInUrl = new URL("/api/auth/signin", req.url);
  // Optional: returnTo parameter to go back after login
  signInUrl.searchParams.set("callbackUrl", req.nextUrl.href);
  return NextResponse.redirect(signInUrl);
}

export const config = {
  matcher: ["/admin/:path*"],
};
