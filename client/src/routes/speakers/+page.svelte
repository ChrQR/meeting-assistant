<script lang="ts">
	import { resolve } from '$app/paths';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';
	import { createGetSpeakers, createSpeaker, type Speaker } from '../../api.js';
	import {
		errorMessage,
		formatDuration,
		initials,
		invalidateByPrefix,
		speakerColor
	} from '#lib/api-helpers.js';
	import AssignSpeakerDialog from '#lib/components/assign-speaker-dialog.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { Avatar, AvatarFallback } from '#lib/components/ui/avatar/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { cn } from '#lib/utils.js';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import UserRoundPlusIcon from '@lucide/svelte/icons/user-round-plus';
	import Loader2Icon from '@lucide/svelte/icons/loader-2';
	import Building2Icon from '@lucide/svelte/icons/building-2';

	const queryClient = useQueryClient();
	const speakersQuery = createGetSpeakers();

	const speakers = $derived(speakersQuery.data?.status === 200 ? speakersQuery.data.data : []);
	const unknownSpeakers = $derived(speakers.filter((s) => s.name === null));
	const knownSpeakers = $derived(
		speakers
			.filter((s) => s.name !== null)
			.sort((a, b) => (a.name ?? '').localeCompare(b.name ?? ''))
	);

	// --- Assign dialog (unknown speakers) ---
	let assignOpen = $state(false);
	let assignSpeaker: Speaker | null = $state(null);

	function openAssign(speaker: Speaker) {
		assignSpeaker = speaker;
		assignOpen = true;
	}

	// --- Create speaker dialog ---
	let createOpen = $state(false);
	let newName = $state('');
	let newEmail = $state('');

	const createSpeakerMutation = createMutation(() => ({
		mutationFn: (variables: { name: string; email: string | null }) =>
			createSpeaker({ name: variables.name, email: variables.email }),
		onSuccess: async () => {
			toast.success('Speaker created');
			await invalidateByPrefix(queryClient, '/speakers');
			createOpen = false;
			newName = '';
			newEmail = '';
		},
		onError: (error: unknown) => {
			toast.error('Could not create speaker', { description: errorMessage(error) });
		}
	}));

	function handleCreate(event: SubmitEvent) {
		event.preventDefault();
		if (!newName.trim() || createSpeakerMutation.isPending) return;
		createSpeakerMutation.mutate({ name: newName.trim(), email: newEmail.trim() || null });
	}

	function handleCreateOpenChange(open: boolean) {
		createOpen = open;
		if (!open) {
			newName = '';
			newEmail = '';
		}
	}
</script>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-6 p-6 lg:p-8">
	<div class="flex flex-wrap items-end justify-between gap-4">
		<div class="flex flex-col gap-1">
			<h1 class="text-2xl font-semibold tracking-tight">Speakers</h1>
			<p class="text-sm text-muted-foreground">
				{#if speakersQuery.isPending}
					Loading speakers…
				{:else}
					{knownSpeakers.length} known
					{#if unknownSpeakers.length > 0}
						· {unknownSpeakers.length} unknown
					{/if}
				{/if}
			</p>
		</div>
		<Button onclick={() => (createOpen = true)}>
			<PlusIcon />
			New speaker
		</Button>
	</div>

	{#if speakersQuery.isError}
		<Card.Root>
			<Card.Content class="text-sm text-destructive">
				Failed to load speakers. Make sure the API is running and try again.
			</Card.Content>
		</Card.Root>
	{:else if speakersQuery.isPending}
		<Card.Root class="gap-0 py-0">
			<div class="flex flex-col divide-y">
				{#each Array.from({ length: 5 }, (_, i) => i) as i (i)}
					<div class="flex items-center gap-4 p-4">
						<Skeleton class="size-9 rounded-full" />
						<div class="flex flex-1 flex-col gap-2">
							<Skeleton class="h-4 w-1/4" />
							<Skeleton class="h-3 w-1/6" />
						</div>
					</div>
				{/each}
			</div>
		</Card.Root>
	{:else}
		{#if unknownSpeakers.length > 0}
			<Card.Root class="gap-0 border-amber-500/30 py-0 ring-amber-500/30">
				<Card.Header class="gap-1 border-b border-amber-500/20 bg-amber-500/5 py-4">
					<Card.Title class="flex items-center gap-2 text-amber-700 dark:text-amber-400">
						<TriangleAlertIcon class="size-4" />
						Unknown speakers ({unknownSpeakers.length})
					</Card.Title>
					<Card.Description>
						These voices were detected but not matched to a known person. Assign them to merge their
						segments and voice embeddings.
					</Card.Description>
				</Card.Header>
				<div class="flex flex-col divide-y divide-amber-500/15">
					{#each unknownSpeakers as speaker (speaker.id)}
						<div class="flex flex-wrap items-center gap-3 px-6 py-3">
							<div class="flex min-w-0 flex-1 items-center gap-3">
								<div
									class="flex size-8 shrink-0 items-center justify-center rounded-full bg-amber-500 text-xs font-semibold text-white"
								>
									?
								</div>
								<div class="flex min-w-0 flex-col">
									<span class="text-sm font-medium">Unknown speaker</span>
									<span class="truncate font-mono text-xs text-muted-foreground">
										{speaker.id.slice(0, 8)} · {speaker.segment_count}
										{speaker.segment_count === 1 ? 'segment' : 'segments'} ·
										{formatDuration(speaker.total_speaking_time)}
									</span>
								</div>
							</div>
							<div class="flex items-center gap-2">
								<Button size="sm" variant="outline" onclick={() => openAssign(speaker)}>
									<UserRoundPlusIcon />
									Assign speaker
								</Button>
								<Button
									size="sm"
									variant="ghost"
									href={resolve('/speakers/[id]', { id: speaker.id })}
									class="text-muted-foreground"
								>
									Details
								</Button>
							</div>
						</div>
					{/each}
				</div>
			</Card.Root>
		{/if}

		{#if knownSpeakers.length === 0}
			<Card.Root>
				<Card.Content class="flex flex-col items-center justify-center gap-3 py-16 text-center">
					<div class="flex size-12 items-center justify-center rounded-full bg-muted">
						<UserRoundPlusIcon class="size-6 text-muted-foreground" />
					</div>
					<div class="flex flex-col gap-1">
						<p class="font-medium">No known speakers yet</p>
						<p class="text-sm text-muted-foreground">
							Create a speaker or assign one of the unknown voices above.
						</p>
					</div>
				</Card.Content>
			</Card.Root>
		{:else}
			<Card.Root class="gap-0 py-0">
				<Table.Root>
					<Table.Header>
						<Table.Row class="hover:bg-transparent">
							<Table.Head>Speaker</Table.Head>
							<Table.Head>Company</Table.Head>
							<Table.Head class="text-right">Embeddings</Table.Head>
							<Table.Head class="text-right">Segments</Table.Head>
							<Table.Head class="text-right">Speaking time</Table.Head>
							<Table.Head class="text-right">Conversations</Table.Head>
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#each knownSpeakers as speaker (speaker.id)}
							<Table.Row>
								<Table.Cell>
									<a
										href={resolve('/speakers/[id]', { id: speaker.id })}
										class="flex items-center gap-3"
									>
										<Avatar size="sm" class="size-8">
											<AvatarFallback class={cn('text-white', speakerColor(speaker.id))}>
												{initials(speaker.name)}
											</AvatarFallback>
										</Avatar>
										<span class="flex min-w-0 flex-col">
											<span class="truncate font-medium underline-offset-4 hover:underline">
												{speaker.name}
											</span>
											{#if speaker.email}
												<span class="truncate text-xs text-muted-foreground">
													{speaker.email}
												</span>
											{/if}
										</span>
									</a>
								</Table.Cell>
								<Table.Cell>
									{#if speaker.company_name}
										<a
											href={resolve('/companies')}
											class="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
										>
											<Building2Icon class="size-3.5" />
											{speaker.company_name}
										</a>
									{:else}
										<span class="text-sm text-muted-foreground">—</span>
									{/if}
								</Table.Cell>
								<Table.Cell class="text-right tabular-nums">{speaker.embedding_count}</Table.Cell>
								<Table.Cell class="text-right tabular-nums">{speaker.segment_count}</Table.Cell>
								<Table.Cell class="text-right text-sm whitespace-nowrap text-muted-foreground">
									{formatDuration(speaker.total_speaking_time)}
								</Table.Cell>
								<Table.Cell class="text-right tabular-nums">{speaker.conversation_count}</Table.Cell
								>
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
			</Card.Root>
		{/if}
	{/if}
</div>

<AssignSpeakerDialog
	speaker={assignSpeaker}
	bind:open={assignOpen}
	onOpenChange={(open) => {
		if (!open) assignSpeaker = null;
	}}
/>

<Dialog.Root bind:open={createOpen} onOpenChange={handleCreateOpenChange}>
	<Dialog.Content>
		<form onsubmit={handleCreate} class="flex flex-col gap-6">
			<Dialog.Header>
				<Dialog.Title>New speaker</Dialog.Title>
				<Dialog.Description>
					Create a named speaker. Voice embeddings are added when they are matched during
					transcription.
				</Dialog.Description>
			</Dialog.Header>
			<div class="flex flex-col gap-4">
				<div class="flex flex-col gap-2">
					<Label for="speaker-name">Name</Label>
					<Input id="speaker-name" bind:value={newName} placeholder="Jane Doe" required />
				</div>
				<div class="flex flex-col gap-2">
					<Label for="speaker-email">
						Email <span class="text-muted-foreground">(optional)</span>
					</Label>
					<Input
						id="speaker-email"
						type="email"
						bind:value={newEmail}
						placeholder="jane@company.com"
					/>
				</div>
			</div>
			<Dialog.Footer>
				<Button
					type="button"
					variant="outline"
					onclick={() => handleCreateOpenChange(false)}
					disabled={createSpeakerMutation.isPending}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={!newName.trim() || createSpeakerMutation.isPending}>
					{#if createSpeakerMutation.isPending}
						<Loader2Icon class="animate-spin" />
					{/if}
					Create speaker
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
