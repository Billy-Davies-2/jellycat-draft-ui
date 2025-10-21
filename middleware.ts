import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { getSessionCookie } from "better-auth/cookies";

function resolveBaseUrl(req: NextRequest): URL {
  const envBase =
    process.env.AUTH_URL ||
    process.env.NEXTAUTH_URL ||
    process.env.NEXT_PUBLIC_APP_URL ||
    process.env.PUBLIC_BASE_URL;
  if (envBase) return new URL(envBase);

  const xfProto = req.headers.get("x-forwarded-proto") || "http";
  const xfHost = (req.headers.get("x-forwarded-host") || req.headers.get("host") || req.nextUrl.host || "localhost").split(",")[0]!.trim();
  const [rawHost, rawPort] = xfHost.split(":");
  const hostname = rawHost === "0.0.0.0" ? "localhost" : rawHost;
  const port = rawPort ? `:${rawPort}` : "";
  return new URL(`${xfProto}://${hostname}${port}`);
}

// Protect admin route: only allow users in Authentik group "homelab-admins".
export async function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  if (!pathname.startsWith("/admin")) {
    return NextResponse.next();
  }

  const baseUrl = resolveBaseUrl(req);
  const sessionCookie = getSessionCookie(req);

  // If no session cookie, redirect to sign-in
  if (!sessionCookie) {
    const signInUrl = new URL("/api/auth/sign-in/oauth2", baseUrl);
    const callbackUrl = new URL(req.nextUrl.pathname + req.nextUrl.search, baseUrl);
    signInUrl.searchParams.set("providerId", "authentik");
    signInUrl.searchParams.set("callbackURL", callbackUrl.toString());
    return NextResponse.redirect(signInUrl);
  }

  // Server-side session check to read roles/claims
  const adminGroup = process.env.AUTH_ADMIN_GROUP || "admins";
  try {
    const sessionEndpoint = new URL("/api/auth/session", baseUrl);
    const res = await fetch(sessionEndpoint.toString(), {
      headers: {
        // Forward cookies for session validation
        cookie: req.headers.get("cookie") || "",
      },
      // Avoid caching at the edge for accuracy
      cache: "no-store",
    });

    if (!res.ok) {
      // If session endpoint says unauthenticated, redirect to sign-in
      const signInUrl = new URL("/api/auth/sign-in/oauth2", baseUrl);
      const callbackUrl = new URL(req.nextUrl.pathname + req.nextUrl.search, baseUrl);
      signInUrl.searchParams.set("providerId", "authentik");
      signInUrl.searchParams.set("callbackURL", callbackUrl.toString());
      return NextResponse.redirect(signInUrl);
    }

    const data = await res.json().catch(() => null);

    // Try multiple likely shapes for groups/roles claims
    const groups: unknown =
      data?.user?.groups ??
      data?.session?.user?.groups ??
      data?.claims?.groups ??
      data?.token?.groups ??
      data?.user?.roles ??
      data?.session?.user?.roles;

    const groupsArr = Array.isArray(groups)
      ? groups
      : typeof groups === "string"
      ? groups.split(",").map((s) => s.trim()).filter(Boolean)
      : [];

    if (groupsArr.includes(adminGroup)) {
      return NextResponse.next();
    }

    // Authenticated but not an admin → 403
    return new NextResponse("Forbidden", { status: 403 });
  } catch (_err) {
    // On error, be safe and redirect to sign-in
    const signInUrl = new URL("/api/auth/sign-in/oauth2", baseUrl);
    const callbackUrl = new URL(req.nextUrl.pathname + req.nextUrl.search, baseUrl);
    signInUrl.searchParams.set("providerId", "authentik");
    signInUrl.searchParams.set("callbackURL", callbackUrl.toString());
    return NextResponse.redirect(signInUrl);
  }
}

export const config = {
  matcher: ["/admin/:path*"],
};
