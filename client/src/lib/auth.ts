import { createGetMe } from '../api.ts';
import { startLogin } from '../custom-fetch.ts';

/**
 * Auth state is resolved from the backend session cookie. The frontend never
 * sees the JWT itself, so the only source of truth is GET /auth/me.
 *
 * Returns the query so components can derive `user` reactively:
 *   const auth = createAuth();
 *   const user = $derived(auth.me.data?.status === 200 ? auth.me.data.data : null);
 */
export function createAuth() {
	const me = createGetMe(() => ({
		query: {
			retry: false,
			staleTime: 1000 * 60 * 5
		}
	}));

	return {
		me,
		login: startLogin
	};
}
