import { NextResponse } from "next/server";

export async function GET(req: Request) {
  const url = new URL(req.url);
  const callbackURL = url.searchParams.get("callbackURL") || "/";
  // Redirect to our login handoff page; Better Auth expects POST to /sign-in/social
  const to = new URL("/login", url.origin);
  to.searchParams.set("callbackURL", callbackURL);
  return NextResponse.redirect(to, { status: 307 });
}
