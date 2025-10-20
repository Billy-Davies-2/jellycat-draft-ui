import { describe, it, expect } from "bun:test";
import { authCallbacks } from "../../app/api/auth/options";

function b64url(input: string) {
  return Buffer.from(input).toString("base64url");
}

describe("authCallbacks.jwt", () => {
  it("extracts groups from profile array", async () => {
    const token: any = {};
    const profile: any = { groups: ["homelab-admins", "users"] };
    const out = await authCallbacks.jwt({ token, profile, account: undefined });
    expect(out.groups).toEqual(["homelab-admins", "users"]);
  });

  it("extracts groups from profile string", async () => {
    const token: any = {};
    const profile: any = { group: "homelab-admins" };
    const out = await authCallbacks.jwt({ token, profile, account: undefined });
    expect(out.groups).toEqual(["homelab-admins"]);
  });

  it("falls back to id_token claims", async () => {
    const token: any = {};
    const payload = b64url(JSON.stringify({ groups: ["homelab-admins"] }));
    const id_token = `x.${payload}.y`;
    const account: any = { id_token };
    const out = await authCallbacks.jwt({ token, profile: undefined, account });
    expect(out.groups).toEqual(["homelab-admins"]);
  });

  it("sets isAdmin from boolean claim when AUTH_ADMIN_CLAIM is set without value", async () => {
    const token: any = {};
    const profile: any = { is_admin: true };
    process.env.AUTH_ADMIN_CLAIM = "is_admin";
    delete process.env.AUTH_ADMIN_VALUE;
    const out = await authCallbacks.jwt({ token, profile, account: undefined });
    expect(out.isAdmin).toBe(true);
  });

  it("sets isAdmin from matching value in string claim", async () => {
    const token: any = {};
    const profile: any = { role: "admin" };
    process.env.AUTH_ADMIN_CLAIM = "role";
    process.env.AUTH_ADMIN_VALUE = "admin";
    const out = await authCallbacks.jwt({ token, profile, account: undefined });
    expect(out.isAdmin).toBe(true);
  });

  it("sets isAdmin from inclusion in array claim", async () => {
    const token: any = {};
    const profile: any = { roles: ["user", "admin"] };
    process.env.AUTH_ADMIN_CLAIM = "roles";
    process.env.AUTH_ADMIN_VALUE = "admin";
    const out = await authCallbacks.jwt({ token, profile, account: undefined });
    expect(out.isAdmin).toBe(true);
  });
});

describe("authCallbacks.session", () => {
  it("copies groups from token to session", async () => {
    const session: any = {};
    const token: any = { groups: ["homelab-admins"] };
    const out = await authCallbacks.session({ session, token });
    expect(out.groups).toEqual(["homelab-admins"]);
  });
});
