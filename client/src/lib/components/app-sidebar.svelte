<script lang="ts">
	import type { Snippet } from 'svelte';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import UploadRecordingDialog from '#lib/components/upload-recording-dialog.svelte';
	import { createAuth } from '#lib/auth.js';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { logout } from '../../api.js';
	import { goto } from '$app/navigation';
	import { Avatar, AvatarFallback } from '#lib/components/ui/avatar/index.js';
	import { initials } from '#lib/api-helpers.js';
	import MessagesSquareIcon from '@lucide/svelte/icons/messages-square';
	import UsersIcon from '@lucide/svelte/icons/users';
	import Building2Icon from '@lucide/svelte/icons/building-2';
	import UploadIcon from '@lucide/svelte/icons/upload';
	import AudioLinesIcon from '@lucide/svelte/icons/audio-lines';
	import SunIcon from '@lucide/svelte/icons/sun';
	import MoonIcon from '@lucide/svelte/icons/moon';
	import LogInIcon from '@lucide/svelte/icons/log-in';
	import { toggleMode, mode } from 'mode-watcher';

	const isDark = $derived(mode.current === 'dark');

	function toggleTheme() {
		toggleMode();
	}

	let { children }: { children: Snippet } = $props();

	let uploadOpen = $state(false);
	let mobileNavOpen = $state(false);

	const auth = createAuth();
	const currentUser = $derived(auth.me.data?.status === 200 ? auth.me.data.data : null);
	const isResolving = $derived(auth.me.isPending);
	const queryClient = useQueryClient();

	const logoutMutation = createMutation(() => ({
		mutationFn: () => logout(),
		onSuccess: async () => {
			queryClient.clear();
			await goto(resolve('/'));
		}
	}));

	const pathname = $derived(page.url.pathname);

	const conversationsActive = $derived(
		pathname === '/' || pathname === '/conversations' || pathname.startsWith('/conversations/')
	);
	const speakersActive = $derived(pathname === '/speakers' || pathname.startsWith('/speakers/'));
	const companiesActive = $derived(pathname === '/companies' || pathname.startsWith('/companies/'));

	function openUpload() {
		mobileNavOpen = false;
		uploadOpen = true;
	}
</script>

<Sidebar.Provider class="h-svh overflow-hidden">
	<Sheet.Root bind:open={mobileNavOpen}>
		<Sheet.Content side="left" class="w-64 p-0">
			<Sheet.Header class="sr-only">
				<Sheet.Title>Navigation</Sheet.Title>
				<Sheet.Description>Main application navigation.</Sheet.Description>
			</Sheet.Header>
			<nav class="flex h-full flex-col gap-2 p-2">
				<a
					href={resolve('/')}
					class="flex items-center gap-2 px-1 py-1.5"
					onclick={() => (mobileNavOpen = false)}
				>
					<div
						class="flex aspect-square size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground"
					>
						<AudioLinesIcon class="size-4" />
					</div>
					<span class="text-sm font-semibold">Meeting Assistant</span>
				</a>
				<Sidebar.Menu>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton
							isActive={conversationsActive}
							onclick={() => (mobileNavOpen = false)}
						>
							{#snippet child({ props })}
								<a href={resolve('/')} {...props}>
									<MessagesSquareIcon />
									<span>Conversations</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton isActive={speakersActive} onclick={() => (mobileNavOpen = false)}>
							{#snippet child({ props })}
								<a href={resolve('/speakers')} {...props}>
									<UsersIcon />
									<span>Speakers</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton isActive={companiesActive} onclick={() => (mobileNavOpen = false)}>
							{#snippet child({ props })}
								<a href={resolve('/companies')} {...props}>
									<Building2Icon />
									<span>Companies</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				</Sidebar.Menu>
				<div class="mt-auto">
					<Button variant="outline" class="w-full justify-start" onclick={openUpload}>
						<UploadIcon />
						Upload recording
					</Button>
				</div>
			</nav>
		</Sheet.Content>
	</Sheet.Root>

	<Sidebar.Root collapsible="icon" variant="sidebar">
		<Sidebar.Header>
			<Sidebar.Menu>
				<Sidebar.MenuItem>
					<Sidebar.MenuButton size="lg" tooltipContent="Meeting Assistant">
						{#snippet child({ props })}
							<a href={resolve('/')} {...props} class="flex items-center gap-2">
								<div
									class="flex aspect-square size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground"
								>
									<AudioLinesIcon class="size-4" />
								</div>
								<div class="grid flex-1 text-left text-sm leading-tight">
									<span class="truncate font-semibold">Meeting Assistant</span>
									<span class="truncate text-xs text-muted-foreground">Transcripts & speakers</span>
								</div>
							</a>
						{/snippet}
					</Sidebar.MenuButton>
				</Sidebar.MenuItem>
			</Sidebar.Menu>
		</Sidebar.Header>

		<Sidebar.Content>
			<Sidebar.Group>
				<Sidebar.GroupLabel>Workspace</Sidebar.GroupLabel>
				<Sidebar.Menu>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton isActive={conversationsActive} tooltipContent="Conversations">
							{#snippet child({ props })}
								<a href={resolve('/')} {...props}>
									<MessagesSquareIcon />
									<span>Conversations</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton isActive={speakersActive} tooltipContent="Speakers">
							{#snippet child({ props })}
								<a href={resolve('/speakers')} {...props}>
									<UsersIcon />
									<span>Speakers</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton isActive={companiesActive} tooltipContent="Companies">
							{#snippet child({ props })}
								<a href={resolve('/companies')} {...props}>
									<Building2Icon />
									<span>Companies</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				</Sidebar.Menu>
			</Sidebar.Group>
		</Sidebar.Content>

		<Sidebar.Footer>
			<Sidebar.Menu>
				<Sidebar.MenuItem>
					<Sidebar.MenuButton
						variant="outline"
						tooltipContent="Upload recording"
						onclick={openUpload}
					>
						<UploadIcon />
						<span>Upload recording</span>
					</Sidebar.MenuButton>
				</Sidebar.MenuItem>

				<Sidebar.MenuItem>
					<Sidebar.MenuButton tooltipContent="Toggle theme" onclick={toggleTheme}>
						{#if isDark}
							<SunIcon />
							<span>Light mode</span>
						{:else}
							<MoonIcon />
							<span>Dark mode</span>
						{/if}
					</Sidebar.MenuButton>
				</Sidebar.MenuItem>

				{#if currentUser}
					<Sidebar.MenuItem>
						<Sidebar.MenuButton
							size="lg"
							tooltipContent={currentUser.name ?? currentUser.email}
							onclick={() => logoutMutation.mutate()}
						>
							<Avatar class="size-8 rounded-lg">
								<AvatarFallback class="rounded-lg text-xs">
									{initials(currentUser.name ?? currentUser.email)}
								</AvatarFallback>
							</Avatar>
							<div class="grid flex-1 text-left text-sm leading-tight">
								<span class="truncate font-medium">{currentUser.name ?? currentUser.email}</span>
								<span class="truncate text-xs text-muted-foreground">Sign out</span>
							</div>
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				{:else if !isResolving}
					<Sidebar.MenuItem>
						<Sidebar.MenuButton tooltipContent="Sign in" onclick={() => auth.login()}>
							<LogInIcon />
							<span>Sign in</span>
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				{/if}
			</Sidebar.Menu>
		</Sidebar.Footer>
		<Sidebar.Rail />
	</Sidebar.Root>

	<Sidebar.Inset class="h-svh overflow-hidden">
		<header class="flex h-14 shrink-0 items-center gap-2 border-b px-4">
			<Sidebar.Trigger class="-ml-1" />
			<span class="text-sm font-medium">Meeting Assistant</span>
		</header>
		<main class="min-h-0 flex-1 overflow-auto">
			{@render children()}
		</main>
	</Sidebar.Inset>
</Sidebar.Provider>

<UploadRecordingDialog bind:open={uploadOpen} />
