import { auth, authSchemaReady } from "@/lib/auth";
import { toNextJsHandler } from "better-auth/next-js";

const base = toNextJsHandler(auth);

async function ready() {
	if (authSchemaReady) {
		try { await authSchemaReady; } catch {/* already logged */}
	}
}

export const GET = async (req: Request) => {
	await ready();
	return base.GET(req);
};

export const POST = async (req: Request) => {
	await ready();
	return base.POST(req);
};
