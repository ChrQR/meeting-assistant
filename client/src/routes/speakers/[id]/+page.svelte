<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';
	import {
		createGetCompanies,
		createGetSpeaker,
		deleteSpeaker,
		updateSpeaker,
		type Speaker
	} from '../../../api.js';
	import {
		errorMessage,
		formatDate,
		formatDuration,
		formatTimestamp,
		initials,
		invalidateByPrefix
	} from '#lib/api-helpers.js';
	import { cn } from '#lib/utils.js';
	import AssignSpeakerDialog from '#lib/components/assign-speaker-dialog.svelte';
	import EditSpeakerDialog from '#lib/components/edit-speaker-dialog.svelte';
	import { createAuth } from '#lib/auth.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Avatar, AvatarFallback } from '#lib/components/ui/avatar/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';
	import Building2Icon from '@lucide/svelte/icons/building-2';
	import MailIcon from '@lucide/svelte/icons/mail';
	import MessagesSquareIcon from '@lucide/svelte/icons/messages-square';
	import ClockIcon from '@lucide/svelte/icons/clock';
	import AudioLinesIcon from '@lucide/svelte/icons/audio-lines';
	import CpuIcon from '@lucide/svelte/icons/cpu';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import UserCheckIcon from '@lucide/svelte/icons/user-check';
	import UserRoundPlusIcon from '@lucide/svelte/icons/user-round-plus';
	import ExternalLinkIcon from '@lucide/svelte/icons/external-link';

	const id = $derived(page.params.id ?? '');
	const queryClient = useQueryClient();

	const speakerQuery = createGetSpeaker(() => id);
	const companiesQuery = createGetCompanies();

	const speaker = $derived(speakerQuery.data?.status === 200 ? speakerQuery.data.data : null);
	const companies = $derived(companiesQuery.data?.status === 200 ? companiesQuery.data.data : []);
	const embeddings = $derived(speaker?.embeddings ?? []);
	const segments = $derived(speaker?.segments ?? []);

	const isUnknown = $derived(speaker !== null && speaker.name === null);

	const segmentsByConversation = $derived.by(() => {
		const groups: { id: string; title: string; segments: typeof segments }[] = [];
		for (const segment of segments) {
			let group = groups.find((g) => g.id === segment.conversation_id);
			if (!group) {
				group = { id: segment.conversation_id, title: segment.conversation_title, segments: [] };
				groups.push(group);
			}
			group.segments.push(segment);
		}
		return groups;
	});

	// --- Assign dialog (unknown speakers) ---
	let assignOpen = $state(false);

	// --- Current user ---
	const auth = createAuth();
	const currentUser = $derived(auth.me.data?.status === 200 ? auth.me.data.data : null);

	// --- Link this speaker to the signed-in user ---
	const linkToMeMutation = createMutation(() => ({
		mutationFn: () => {
			if (!speaker || !currentUser) throw new Error('Not signed in');
			return updateSpeaker(speaker.id, { user_id: currentUser.id });
		},
		onSuccess: async () => {
			toast.success('Speaker linked to your account');
			await invalidateByPrefix(queryClient, '/speakers');
		},
		onError: (error: unknown) => {
			toast.error('Could not link speaker', { description: errorMessage(error) });
		}
	}));

	// --- Tabs ---
	let activeTab = $state('segments');
	let editOpen = $state(false);

	// --- Company select ---
	const companyValue = $derived(speaker?.company_id ?? 'none');

	const updateMutation = createMutation(() => ({
		mutationFn: (variables: { id: string; companyId: string | null }) =>
			updateSpeaker(
				variables.id,
				variables.companyId ? { company_id: variables.companyId } : { clear_company: true }
			),
		onSuccess: async () => {
			toast.success('Speaker updated');
			await invalidateByPrefix(queryClient, '/speakers', '/companies');
		},
		onError: (error: unknown) => {
			toast.error('Could not update speaker', { description: errorMessage(error) });
		}
	}));

	function handleCompanyChange(value: string | undefined) {
		if (!speaker || !value) return;
		const companyId = value === 'none' ? null : value;
		if (companyId === speaker.company_id) return;
		updateMutation.mutate({ id: speaker.id, companyId });
	}

	// --- Delete ---
	let deleteRequested = $state(false);

	const deleteMutation = createMutation(() => ({
		mutationFn: (speakerId: string) => deleteSpeaker(speakerId),
		onSuccess: async () => {
			toast.success('Speaker deleted');
			await Promise.all([invalidateByPrefix(queryClient, '/speakers', '/companies')]);
			deleteRequested = false;
			await goto(resolve('/speakers'));
		},
		onError: (error: unknown) => {
			deleteRequested = false;
			const message = errorMessage(error);
			if (/409|conflict|segment/i.test(message)) {
				toast.error('Cannot delete speaker', {
					description: 'This speaker still has segments. Reassign or merge their segments first.'
				});
			} else {
				toast.error('Could not delete speaker', { description: message });
			}
		}
	}));
</script>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-6 p-6 lg:p-8">
	<a
		href={resolve('/speakers')}
		class="inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
	>
		<ArrowLeftIcon class="size-4" />
		All speakers
	</a>

	{#if speakerQuery.isPending}
		<div class="flex items-center gap-4">
			<Skeleton class="size-14 rounded-full" />
			<div class="flex flex-col gap-2">
				<Skeleton class="h-7 w-48" />
				<Skeleton class="h-4 w-32" />
			</div>
		</div>
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			{#each Array.from({ length: 4 }, (_, i) => i) as i (i)}
				<Skeleton class="h-24 w-full rounded-xl" />
			{/each}
		</div>
		<Skeleton class="h-64 w-full rounded-xl" />
	{:else if speakerQuery.isError || !speaker}
		<Card.Root>
			<Card.Content class="text-sm text-destructive">
				Speaker not found or could not be loaded.
			</Card.Content>
		</Card.Root>
	{:else}
		<div class="flex flex-wrap items-start justify-between gap-4">
			<div class="flex min-w-0 items-center gap-4">
				<Avatar size="lg" class="size-14">
					<AvatarFallback
						class={cn(
							'text-lg font-semibold text-white',
							isUnknown ? 'bg-amber-500' : 'bg-primary'
						)}
					>
						{initials(speaker.name)}
					</AvatarFallback>
				</Avatar>
				<div class="flex min-w-0 flex-col gap-1">
					<h1 class="truncate text-2xl font-semibold tracking-tight">
						{speaker.name ?? 'Unknown speaker'}
					</h1>
					<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
						{#if speaker.email}
							<span class="inline-flex items-center gap-1">
								<MailIcon class="size-3.5" />
								{speaker.email}
							</span>
						{/if}
						<span>Added {formatDate(speaker.created_at)}</span>
					</div>
				</div>
			</div>
			<div class="flex items-center gap-2">
				{#if isUnknown}
					<Button onclick={() => (assignOpen = true)}>
						<UserRoundPlusIcon />
						Assign speaker
					</Button>
				{:else}
					{#if currentUser && !speaker.user_id}
						<Button
							variant="outline"
							onclick={() => linkToMeMutation.mutate()}
							disabled={linkToMeMutation.isPending}
						>
							<UserCheckIcon />
							This is me
						</Button>
					{/if}
					<Button variant="outline" onclick={() => (editOpen = true)}>
						<PencilIcon />
						Edit speaker
					</Button>
				{/if}
				<Button variant="destructive" onclick={() => (deleteRequested = true)}>
					<Trash2Icon />
					Delete
				</Button>
			</div>
		</div>

		<div class="flex flex-wrap items-center gap-3">
			<span class="inline-flex items-center gap-1.5 text-sm text-muted-foreground">
				<Building2Icon class="size-4" />
				Company
			</span>
			<Select.Root
				type="single"
				value={companyValue}
				onValueChange={handleCompanyChange}
				disabled={updateMutation.isPending}
			>
				<Select.Trigger class="w-56 justify-between">
					{#if speaker.company_name}
						<span class="truncate">{speaker.company_name}</span>
					{:else}
						<span class="truncate text-muted-foreground">No company</span>
					{/if}
				</Select.Trigger>
				<Select.Content>
					<Select.Item value="none" label="No company" />
					{#each companies as company (company.id)}
						<Select.Item value={company.id} label={company.name} />
					{/each}
				</Select.Content>
			</Select.Root>
			<a
				href={resolve('/companies')}
				class="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
			>
				Manage companies
			</a>
		</div>

		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<Card.Root size="sm">
				<Card.Content class="gap-1">
					<span class="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
						<MessagesSquareIcon class="size-3.5" />
						Conversations
					</span>
					<span class="text-2xl font-semibold tabular-nums">{speaker.conversation_count}</span>
				</Card.Content>
			</Card.Root>
			<Card.Root size="sm">
				<Card.Content class="gap-1">
					<span class="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
						<AudioLinesIcon class="size-3.5" />
						Segments
					</span>
					<span class="text-2xl font-semibold tabular-nums">{speaker.segment_count}</span>
				</Card.Content>
			</Card.Root>
			<Card.Root size="sm">
				<Card.Content class="gap-1">
					<span class="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
						<ClockIcon class="size-3.5" />
						Speaking time
					</span>
					<span class="text-2xl font-semibold tabular-nums">
						{formatDuration(speaker.total_speaking_time)}
					</span>
				</Card.Content>
			</Card.Root>
			<Card.Root size="sm">
				<Card.Content class="gap-1">
					<span class="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
						<CpuIcon class="size-3.5" />
						Voice embeddings
					</span>
					<span class="text-2xl font-semibold tabular-nums">{speaker.embedding_count}</span>
				</Card.Content>
			</Card.Root>
		</div>

		<Tabs.Root bind:value={activeTab} class="gap-4">
			<Tabs.List>
				<Tabs.Trigger value="segments">
					<AudioLinesIcon />
					Segments ({segments.length})
				</Tabs.Trigger>
				<Tabs.Trigger value="embeddings">
					<CpuIcon />
					Voice embeddings ({embeddings.length})
				</Tabs.Trigger>
			</Tabs.List>

			<Tabs.Content value="segments">
				{#if segmentsByConversation.length === 0}
					<Card.Root>
						<Card.Content class="py-12 text-center text-sm text-muted-foreground">
							This speaker has no transcript segments yet.
						</Card.Content>
					</Card.Root>
				{:else}
					<div class="flex flex-col gap-4">
						{#each segmentsByConversation as group (group.id)}
							<Card.Root class="gap-0 py-0">
								<Card.Header class="gap-1.5 border-b py-3">
									<a
										href={resolve('/conversations/[id]', { id: group.id })}
										class="flex items-center gap-1.5 font-medium underline-offset-4 hover:underline"
									>
										{group.title || 'Untitled conversation'}
										<ExternalLinkIcon class="size-3.5 text-muted-foreground" />
									</a>
									<Card.Description>
										{group.segments.length}
										{group.segments.length === 1 ? 'segment' : 'segments'}
									</Card.Description>
								</Card.Header>
								<div class="flex flex-col divide-y">
									{#each group.segments as segment (segment.id)}
										<div class="flex gap-3 px-4 py-2.5">
											<a
												href={resolve('/conversations/[id]', { id: group.id })}
												class="shrink-0 font-mono text-xs text-muted-foreground tabular-nums underline-offset-2 hover:text-foreground hover:underline"
											>
												{formatTimestamp(segment.start_time)}
											</a>
											<p class="text-sm leading-relaxed">{segment.text}</p>
										</div>
									{/each}
								</div>
							</Card.Root>
						{/each}
					</div>
				{/if}
			</Tabs.Content>

			<Tabs.Content value="embeddings">
				<Card.Root class="gap-0 py-0">
					<Card.Header class="gap-1.5 border-b py-4">
						<Card.Title>Voice embeddings</Card.Title>
						<Card.Description>
							Each embedding is a voice-print vector computed from this speaker's audio. They are
							used to recognize the speaker in future recordings.
						</Card.Description>
					</Card.Header>
					{#if embeddings.length === 0}
						<Card.Content class="py-12 text-center text-sm text-muted-foreground">
							No voice embeddings yet.
						</Card.Content>
					{:else}
						<div class="flex flex-col divide-y">
							{#each embeddings as embedding (embedding.id)}
								<div class="flex items-center justify-between gap-4 px-6 py-3">
									<span class="font-mono text-xs">{embedding.id}</span>
									<span class="text-xs whitespace-nowrap text-muted-foreground">
										{formatDate(embedding.created_at)}
									</span>
								</div>
							{/each}
						</div>
					{/if}
				</Card.Root>
			</Tabs.Content>
		</Tabs.Root>

		<Separator />
		<p class="text-xs text-muted-foreground">
			Speaker ID <span class="font-mono">{speaker.id}</span>
		</p>
	{/if}
</div>

<EditSpeakerDialog bind:open={editOpen} {speaker} />

<AssignSpeakerDialog
	speaker={speaker as Speaker | null}
	bind:open={assignOpen}
	onAssigned={() => goto(resolve('/speakers'))}
/>

<AlertDialog.Root bind:open={() => deleteRequested, (value) => (deleteRequested = value)}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>Delete this speaker?</AlertDialog.Title>
			<AlertDialog.Description>
				{#if speaker}
					<strong>{speaker.name ?? 'This speaker'}</strong> and their voice embeddings will be permanently
					removed. Speakers that still have transcript segments cannot be deleted.
				{/if}
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel disabled={deleteMutation.isPending}>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action
				variant="destructive"
				disabled={deleteMutation.isPending}
				onclick={() => {
					if (speaker) deleteMutation.mutate(speaker.id);
				}}
			>
				Delete speaker
			</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
