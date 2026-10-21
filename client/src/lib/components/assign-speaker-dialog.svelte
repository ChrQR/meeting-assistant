<script lang="ts">
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';
	import {
		assignSpeaker,
		createGetSpeakers,
		type AssignSpeakerBody,
		type Speaker
	} from '../../api.js';
	import { errorMessage, formatDuration, invalidateByPrefix } from '#lib/api-helpers.js';
	import Loader2Icon from '@lucide/svelte/icons/loader-2';
	import UserRoundCheckIcon from '@lucide/svelte/icons/user-round-check';
	import UserRoundPlusIcon from '@lucide/svelte/icons/user-round-plus';

	let {
		speaker = null,
		open = $bindable(false),
		onOpenChange = () => {},
		onAssigned
	}: {
		speaker?: Speaker | null;
		open?: boolean;
		onOpenChange?: (open: boolean) => void;
		onAssigned?: (speaker: Speaker) => void;
	} = $props();

	const queryClient = useQueryClient();
	const speakersQuery = createGetSpeakers();

	const knownSpeakers = $derived(
		(speakersQuery.data?.status === 200 ? speakersQuery.data.data : []).filter(
			(s) => s.name !== null && s.id !== speaker?.id
		)
	);

	let mode = $state<'existing' | 'new'>('existing');
	let targetId = $state<string>('');
	let newName = $state('');
	let newEmail = $state('');

	const canSubmit = $derived(mode === 'existing' ? targetId !== '' : newName.trim() !== '');

	const mutation = createMutation(() => ({
		mutationFn: (body: { id: string; payload: AssignSpeakerBody }) =>
			assignSpeaker(body.id, body.payload),
		onSuccess: async (result) => {
			toast.success('Speaker assigned', {
				description: result.data?.name
					? `Segments and voice embeddings are now attributed to ${result.data.name}.`
					: undefined
			});
			// Prefix matching also refreshes `/speakers/:id` and `/conversations/:id`.
			await Promise.all([invalidateByPrefix(queryClient, '/speakers', '/conversations')]);
			if (result.data) onAssigned?.(result.data);
			handleOpenChange(false);
		},
		onError: (error: unknown) => {
			toast.error('Could not assign speaker', { description: errorMessage(error) });
		}
	}));

	function reset() {
		mode = 'existing';
		targetId = '';
		newName = '';
		newEmail = '';
	}

	function handleOpenChange(value: boolean) {
		open = value;
		onOpenChange(value);
		if (!value) reset();
	}

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		if (!speaker || !canSubmit || mutation.isPending) return;

		const payload: AssignSpeakerBody =
			mode === 'existing'
				? { target_speaker_id: targetId }
				: { name: newName.trim(), email: newEmail.trim() || null };

		mutation.mutate({ id: speaker.id, payload });
	}
</script>

<Dialog.Root bind:open onOpenChange={handleOpenChange}>
	<Dialog.Content>
		<form onsubmit={handleSubmit} class="flex flex-col gap-6">
			<Dialog.Header>
				<Dialog.Title>Assign speaker</Dialog.Title>
				<Dialog.Description>
					{#if speaker}
						This speaker has {speaker.segment_count}
						{speaker.segment_count === 1 ? 'segment' : 'segments'}. Merging will move their segments
						and voice embeddings to the selected speaker.
					{:else}
						Give this unknown speaker a name or merge them into an existing speaker.
					{/if}
				</Dialog.Description>
			</Dialog.Header>

			{#if speaker}
				<div
					class="flex items-center justify-between rounded-md border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-sm"
				>
					<span class="text-muted-foreground">Unknown speaker</span>
					<span class="font-mono text-xs text-muted-foreground">{speaker.id.slice(0, 8)}</span>
				</div>
			{/if}

			<Tabs.Root bind:value={mode} class="gap-4">
				<Tabs.List class="w-full">
					<Tabs.Trigger value="existing" class="flex-1">
						<UserRoundCheckIcon />
						Existing speaker
					</Tabs.Trigger>
					<Tabs.Trigger value="new" class="flex-1">
						<UserRoundPlusIcon />
						New name
					</Tabs.Trigger>
				</Tabs.List>

				<Tabs.Content value="existing" class="flex flex-col gap-2">
					{#if speakersQuery.isPending}
						<Skeleton class="h-9 w-full" />
					{:else if knownSpeakers.length === 0}
						<p class="text-sm text-muted-foreground">
							There are no other named speakers yet. Use
							<strong>New name</strong> to name this speaker instead.
						</p>
					{:else}
						<Select.Root type="single" bind:value={targetId}>
							<Select.Trigger class="w-full justify-between">
								{#if targetId}
									{@const selected = knownSpeakers.find((s) => s.id === targetId)}
									<span class="truncate">{selected?.name ?? targetId}</span>
								{:else}
									<Select.Value placeholder="Choose a speaker…" />
								{/if}
							</Select.Trigger>
							<Select.Content>
								{#each knownSpeakers as known (known.id)}
									<Select.Item value={known.id} label={known.name ?? known.id}>
										<span class="truncate">{known.name}</span>
										<span class="shrink-0 truncate text-xs text-muted-foreground">
											{known.segment_count} seg · {formatDuration(known.total_speaking_time)}
										</span>
									</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					{/if}
					<p class="text-xs text-muted-foreground">
						This unknown speaker will be merged into the selected speaker and then removed.
					</p>
				</Tabs.Content>

				<Tabs.Content value="new" class="flex flex-col gap-4">
					<div class="flex flex-col gap-2">
						<Label for="assign-name">Name</Label>
						<Input
							id="assign-name"
							bind:value={newName}
							placeholder="Jane Doe"
							autocomplete="off"
							required={mode === 'new'}
						/>
					</div>
					<div class="flex flex-col gap-2">
						<Label for="assign-email">
							Email <span class="text-muted-foreground">(optional)</span>
						</Label>
						<Input
							id="assign-email"
							type="email"
							bind:value={newEmail}
							placeholder="jane@company.com"
							autocomplete="off"
						/>
					</div>
					<p class="text-xs text-muted-foreground">
						The unknown speaker keeps their voice embeddings and segments, they are just given a
						name.
					</p>
				</Tabs.Content>
			</Tabs.Root>

			<Dialog.Footer>
				<Button
					type="button"
					variant="outline"
					onclick={() => handleOpenChange(false)}
					disabled={mutation.isPending}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={!canSubmit || mutation.isPending}>
					{#if mutation.isPending}
						<Loader2Icon class="animate-spin" />
						Saving…
					{:else if mode === 'existing'}
						Merge speaker
					{:else}
						Assign name
					{/if}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
