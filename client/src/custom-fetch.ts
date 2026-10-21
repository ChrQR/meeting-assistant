const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';

/** Guards against firing several login redirects at once. */
let redirecting = false;

/** Raised for 401 responses so callers can redirect to the sign-in flow. */
export class UnauthorizedError extends Error {
	constructor() {
		super('Not authenticated');
		this.name = 'UnauthorizedError';
	}
}

/**
 * Sends an API request. Credentials are included so the httpOnly session cookie
 * travels with every call; the token is never readable from JavaScript.
 */
export const customFetch = async <T>(url: string, options: RequestInit = {}): Promise<T> => {
	const fullUrl = `${BASE_URL}${url}`;
	const res = await fetch(fullUrl, {
		...options,
		credentials: 'include'
	});

	if (res.status === 401) {
		// Session expired or absent: kick off the SSO flow once, unless we are
		// already probing auth state (that request is expected to 401).
		if (!url.startsWith('/auth/me') && !redirecting) {
			redirecting = true;
			startLogin();
		}
		throw new UnauthorizedError();
	}

	if (!res.ok) {
		const errorText = await res.text().catch(() => '');
		throw new Error(errorText || `Request failed with status ${res.status}`);
	}

	const body = [204, 205, 304].includes(res.status) ? null : await res.text();
	const parsedData = body ? JSON.parse(body) : undefined;

	return {
		data: parsedData,
		status: res.status,
		headers: res.headers
	} as T;
};

/** Starts the SSO flow by navigating to the backend login endpoint. */
export function startLogin(provider = 'microsoft'): void {
	window.location.href = `${BASE_URL}/auth/login?provider=${encodeURIComponent(provider)}`;
}
