<script lang="ts">
	import { resolve } from '$app/paths';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';
	import {
		createCompany,
		createGetCompanies,
		createGetSpeakers,
		deleteCompany,
		updateCompany,
		type Company
	} from '../../api.js';
	import { errorMessage, formatDate, initials, invalidateByPrefix } from '#lib/api-helpers.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Collapsible from '#lib/components/ui/collapsible/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { Avatar, AvatarFallback } from '#lib/components/ui/avatar/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import Building2Icon from '@lucide/svelte/icons/building-2';
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import PencilIcon from '@lucide/svelte/icons/pencil';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import UsersIcon from '@lucide/svelte/icons/users';
	import Loader2Icon from '@lucide/svelte/icons/loader-2';

	const queryClient = useQueryClient();
	const companiesQuery = createGetCompanies();
	const speakersQuery = createGetSpeakers();

	const companies = $derived(companiesQuery.data?.status === 200 ? companiesQuery.data.data : []);
	const speakers = $derived(speakersQuery.data?.status === 200 ? speakersQuery.data.data : []);

	function speakersFor(companyId: string) {
		return speakers.filter((s) => s.company_id === companyId);
	}

	async function invalidateAll() {
		await Promise.all([invalidateByPrefix(queryClient, '/companies', '/speakers')]);
	}

	// --- Create / rename dialog (shared) ---
	let editorOpen = $state(false);
	let editing: Company | null = $state(null);
	let name = $state('');

	function openCreate() {
		editing = null;
		name = '';
		editorOpen = true;
	}

	function openRename(company: Company) {
		editing = company;
		name = company.name;
		editorOpen = true;
	}

	function handleEditorOpenChange(open: boolean) {
		editorOpen = open;
		if (!open) {
			editing = null;
			name = '';
		}
	}

	const saveMutation = createMutation(() => ({
		mutationFn: async (variables: { id: string | null; name: string }) => {
			if (variables.id) {
				await updateCompany(variables.id, { name: variables.name });
			} else {
				await createCompany({ name: variables.name });
			}
		},
		onSuccess: async (_data, variables) => {
			toast.success(variables.id ? 'Company renamed' : 'Company created');
			await invalidateAll();
			handleEditorOpenChange(false);
		},
		onError: (error: unknown) => {
			toast.error(editing ? 'Could not rename company' : 'Could not create company', {
				description: errorMessage(error)
			});
		}
	}));

	function handleSave(event: SubmitEvent) {
		event.preventDefault();
		const trimmed = name.trim();
		if (!trimmed || saveMutation.isPending) return;
		saveMutation.mutate({ id: editing?.id ?? null, name: trimmed });
	}

	// --- Delete ---
	let deleteTarget: Company | null = $state(null);
	const deleteOpen = $derived(deleteTarget !== null);

	const deleteMutation = createMutation(() => ({
		mutationFn: (id: string) => deleteCompany(id),
		onSuccess: async (_data, id) => {
			const count = speakersFor(id).length;
			toast.success('Company deleted', {
				description:
					count > 0
						? `${count} ${count === 1 ? 'speaker was' : 'speakers were'} detached.`
						: undefined
			});
			await invalidateAll();
			deleteTarget = null;
		},
		onError: (error: unknown) => {
			toast.error('Could not delete company', { description: errorMessage(error) });
		}
	}));
</script>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-6 p-6 lg:p-8">
	<div class="flex flex-wrap items-end justify-between gap-4">
		<div class="flex flex-col gap-1">
			<h1 class="text-2xl font-semibold tracking-tight">Companies</h1>
			<p class="text-sm text-muted-foreground">
				{#if companiesQuery.isPending}
					Loading companies…
				{:else}
					{companies.length}
					{companies.length === 1 ? 'company' : 'companies'}
				{/if}
			</p>
		</div>
		<Button onclick={openCreate}>
			<PlusIcon />
			New company
		</Button>
	</div>

	{#if companiesQuery.isError}
		<Card.Root>
			<Card.Content class="text-sm text-destructive">
				Failed to load companies. Make sure the API is running and try again.
			</Card.Content>
		</Card.Root>
	{:else if companiesQuery.isPending}
		<div class="grid gap-4 md:grid-cols-2">
			{#each Array.from({ length: 4 }, (_, i) => i) as i (i)}
				<Skeleton class="h-32 w-full rounded-xl" />
			{/each}
		</div>
	{:else if companies.length === 0}
		<Card.Root>
			<Card.Content class="flex flex-col items-center justify-center gap-3 py-16 text-center">
				<div class="flex size-12 items-center justify-center rounded-full bg-muted">
					<Building2Icon class="size-6 text-muted-foreground" />
				</div>
				<div class="flex flex-col gap-1">
					<p class="font-medium">No companies yet</p>
					<p class="text-sm text-muted-foreground">Group speakers by the company they belong to.</p>
				</div>
				<Button class="mt-2" onclick={openCreate}>
					<PlusIcon />
					New company
				</Button>
			</Card.Content>
		</Card.Root>
	{:else}
		<div class="grid gap-4 md:grid-cols-2">
			{#each companies as company (company.id)}
				{@const companySpeakers = speakersFor(company.id)}
				<Collapsible.Root class="group">
					<Card.Root class="h-full gap-0 py-0">
						<Card.Header class="flex-row items-start justify-between gap-2 py-4">
							<div class="flex min-w-0 items-start gap-3">
								<div class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted">
									<Building2Icon class="size-4 text-muted-foreground" />
								</div>
								<div class="flex min-w-0 flex-col gap-0.5">
									<Card.Title class="truncate">{company.name}</Card.Title>
									<Card.Description>
										{company.speaker_count}
										{company.speaker_count === 1 ? 'speaker' : 'speakers'} · added
										{formatDate(company.created_at)}
									</Card.Description>
								</div>
							</div>
							<div class="flex shrink-0 items-center gap-1">
								<Button
									variant="ghost"
									size="icon-sm"
									aria-label="Rename company"
									onclick={() => openRename(company)}
								>
									<PencilIcon />
								</Button>
								<Button
									variant="ghost"
									size="icon-sm"
									aria-label="Delete company"
									class="text-destructive hover:text-destructive"
									onclick={() => (deleteTarget = company)}
								>
									<Trash2Icon />
								</Button>
							</div>
						</Card.Header>

						<Card.Content class="pb-3">
							{#if companySpeakers.length > 0}
								<Collapsible.Content>
									<ul class="flex flex-col gap-1">
										{#each companySpeakers as speaker (speaker.id)}
											<li>
												<a
													href={resolve('/speakers/[id]', { id: speaker.id })}
													class="-mx-2 flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-muted/60"
												>
													<Avatar size="sm" class="size-6">
														<AvatarFallback class="text-[10px]">
															{initials(speaker.name)}
														</AvatarFallback>
													</Avatar>
													<span class="truncate text-sm">
														{speaker.name ?? 'Unknown speaker'}
													</span>
													{#if speaker.email}
														<span class="ml-auto truncate text-xs text-muted-foreground">
															{speaker.email}
														</span>
													{/if}
												</a>
											</li>
										{/each}
									</ul>
								</Collapsible.Content>
								<Collapsible.Trigger
									class="mt-1 -ml-2 inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-muted-foreground hover:text-foreground"
								>
									<ChevronDownIcon
										class="size-3.5 transition-transform group-data-[state=open]:rotate-180"
									/>
									{companySpeakers.length}
									{companySpeakers.length === 1 ? 'speaker' : 'speakers'}
								</Collapsible.Trigger>
							{:else}
								<p class="flex items-center gap-1.5 text-xs text-muted-foreground">
									<UsersIcon class="size-3.5" />
									No speakers assigned yet.
								</p>
							{/if}
						</Card.Content>
					</Card.Root>
				</Collapsible.Root>
			{/each}
		</div>
	{/if}
</div>

<Dialog.Root bind:open={editorOpen} onOpenChange={handleEditorOpenChange}>
	<Dialog.Content>
		<form onsubmit={handleSave} class="flex flex-col gap-6">
			<Dialog.Header>
				<Dialog.Title>{editing ? 'Rename company' : 'New company'}</Dialog.Title>
				<Dialog.Description>
					{editing
						? 'Update the display name of this company.'
						: 'Create a company to group related speakers.'}
				</Dialog.Description>
			</Dialog.Header>
			<div class="flex flex-col gap-2">
				<Label for="company-name">Name</Label>
				<Input id="company-name" bind:value={name} placeholder="Acme Inc." required autofocus />
			</div>
			<Dialog.Footer>
				<Button
					type="button"
					variant="outline"
					onclick={() => handleEditorOpenChange(false)}
					disabled={saveMutation.isPending}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={!name.trim() || saveMutation.isPending}>
					{#if saveMutation.isPending}
						<Loader2Icon class="animate-spin" />
					{/if}
					{editing ? 'Save changes' : 'Create company'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>

<AlertDialog.Root
	bind:open={
		() => deleteOpen,
		(value) => {
			if (!value) deleteTarget = null;
		}
	}
>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>Delete company?</AlertDialog.Title>
			<AlertDialog.Description>
				{#if deleteTarget}
					<strong>{deleteTarget.name}</strong> will be removed.
					{deleteTarget.speaker_count > 0
						? `${deleteTarget.speaker_count} ${deleteTarget.speaker_count === 1 ? 'speaker will' : 'speakers will'} be detached but not deleted.`
						: 'No speakers are attached.'}
				{/if}
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel disabled={deleteMutation.isPending}>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action
				variant="destructive"
				disabled={deleteMutation.isPending}
				onclick={() => {
					if (deleteTarget) deleteMutation.mutate(deleteTarget.id);
				}}
			>
				Delete company
			</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
