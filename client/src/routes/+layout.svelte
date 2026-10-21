<script lang="ts">
	import './layout.css';
	import favicon from '#lib/assets/favicon.svg';
	import type { LayoutProps } from './$types';
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import { Toaster } from '#lib/components/ui/sonner/index.js';
	import AppSidebar from '#lib/components/app-sidebar.svelte';
	import { ModeWatcher } from 'mode-watcher';
	import { toast } from 'svelte-sonner';

	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				staleTime: 1000 * 60,
				retry: 1
			}
		}
	});

	let { children }: LayoutProps = $props();

	// Surface an error passed back from the OAuth callback, then strip it so a
	// refresh does not repeat the toast.
	const initialParams = new URL(window.location.href).searchParams;
	const authError = initialParams.get('auth_error');
	if (authError) {
		toast.error('Sign-in failed', { description: authError });
		initialParams.delete('auth_error');
		const remaining = initialParams.toString();
		window.history.replaceState(
			{},
			'',
			window.location.pathname + (remaining ? `?${remaining}` : '')
		);
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Meeting Assistant</title>
</svelte:head>

<ModeWatcher defaultMode="system" />
<QueryClientProvider client={queryClient}>
	<AppSidebar>
		{@render children()}
	</AppSidebar>
	<Toaster />
</QueryClientProvider>
