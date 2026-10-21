<script lang="ts">
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';
	import {
		createGetCompanies,
		updateSpeaker,
		type Speaker,
		type UpdateSpeakerBody
	} from '../../api.js';
	import { errorMessage, invalidateByPrefix } from '#lib/api-helpers.js';
	import Loader2Icon from '@lucide/svelte/icons/loader-2';

	let {
		speaker,
		open = $bindable(false),
		onOpenChange = () => {},
		onSaved
	}: {
		speaker: Speaker | null;
		open?: boolean;
		onOpenChange?: (open: boolean) => void;
		onSaved?: (speaker: Speaker) => void;
	} = $props();

	const queryClient = useQueryClient();
	const companiesQuery = createGetCompanies();

	const companies = $derived(
		companiesQuery.data?.status === 200 ? (companiesQuery.data.data ?? []) : []
	);

	/** `none` is the sentinel for "no company" in the select. */
	const NO_COMPANY = 'none';

	let name = $state('');
	let email = $state('');
	let companyId = $state(NO_COMPANY);

	// Sync form fields whenever the dialog opens for a (possibly different) speaker.
	let syncedFor = $state<string | null>(null);
	$effect(() => {
		if (!open || !speaker) return;
		if (syncedFor === speaker.id) return;
		syncedFor = speaker.id;
		name = speaker.name ?? '';
		email = speaker.email ?? '';
		companyId = speaker.company_id ?? NO_COMPANY;
	});
	$effect(() => {
		if (!open) syncedFor = null;
	});

	const canSubmit = $derived(name.trim() !== '');

	const mutation = createMutation(() => ({
		mutationFn: (body: { id: string; payload: UpdateSpeakerBody }) =>
			updateSpeaker(body.id, body.payload),
		onSuccess: async (result) => {
			toast.success('Speaker updated');
			await invalidateByPrefix(queryClient, '/speakers', '/companies', '/conversations');
			if (result.data) onSaved?.(result.data);
			handleOpenChange(false);
		},
		onError: (error: unknown) => {
			toast.error('Could not update speaker', { description: errorMessage(error) });
		}
	}));

	function handleOpenChange(value: boolean) {
		open = value;
		onOpenChange(value);
	}

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		if (!speaker || !canSubmit || mutation.isPending) return;

		const payload: UpdateSpeakerBody = { name: name.trim() };

		// Only send email when it changed, so clearing it is explicit.
		const trimmedEmail = email.trim();
		if (trimmedEmail !== (speaker.email ?? '')) {
			payload.email = trimmedEmail === '' ? null : trimmedEmail;
		}

		const targetCompany = companyId === NO_COMPANY ? null : companyId;
		if (targetCompany !== speaker.company_id) {
			if (targetCompany === null) {
				payload.clear_company = true;
			} else {
				payload.company_id = targetCompany;
			}
		}

		mutation.mutate({ id: speaker.id, payload });
	}
</script>

<Dialog.Root bind:open onOpenChange={handleOpenChange}>
	<Dialog.Content>
		<form onsubmit={handleSubmit} class="flex flex-col gap-6">
			<Dialog.Header>
				<Dialog.Title>Edit speaker</Dialog.Title>
				<Dialog.Description>
					Update the name, email and company attributed to this voice.
				</Dialog.Description>
			</Dialog.Header>

			<div class="flex flex-col gap-4">
				<div class="flex flex-col gap-2">
					<Label for="edit-speaker-name">Name</Label>
					<Input
						id="edit-speaker-name"
						bind:value={name}
						placeholder="Jane Doe"
						autocomplete="off"
						required
					/>
				</div>

				<div class="flex flex-col gap-2">
					<Label for="edit-speaker-email">
						Email <span class="text-muted-foreground">(optional)</span>
					</Label>
					<Input
						id="edit-speaker-email"
						type="email"
						bind:value={email}
						placeholder="jane@company.com"
						autocomplete="off"
					/>
				</div>

				<div class="flex flex-col gap-2">
					<Label for="edit-speaker-company">Company</Label>
					<Select.Root type="single" bind:value={companyId}>
						<Select.Trigger id="edit-speaker-company" class="w-full justify-between">
							{#if companyId === NO_COMPANY}
								<span class="truncate text-muted-foreground">No company</span>
							{:else}
								{@const selected = companies.find((company) => company.id === companyId)}
								<span class="truncate">{selected?.name ?? speaker?.company_name ?? companyId}</span>
							{/if}
						</Select.Trigger>
						<Select.Content>
							<Select.Item value={NO_COMPANY} label="No company" />
							{#each companies as company (company.id)}
								<Select.Item value={company.id} label={company.name} />
							{/each}
						</Select.Content>
					</Select.Root>
				</div>
			</div>

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
					{:else}
						Save changes
					{/if}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
