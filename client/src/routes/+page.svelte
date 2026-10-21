<script lang="ts">
	import { resolve } from '$app/paths';
	import { createGetConversations, createGetRecordings } from '../api.js';
	import { conversationDate, formatDuration, isProcessing } from '#lib/api-helpers.js';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import UploadRecordingDialog from '#lib/components/upload-recording-dialog.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import MessagesSquareIcon from '@lucide/svelte/icons/messages-square';
	import UploadIcon from '@lucide/svelte/icons/upload';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import ClockIcon from '@lucide/svelte/icons/clock';
	import Loader2Icon from '@lucide/svelte/icons/loader-2';

	/** While any recording is processing, refresh so the UI is not stale. */
	const PROCESSING_POLL_MS = 3000;

	const recordingsQuery = createGetRecordings(() => ({
		query: {
			refetchInterval: (query) =>
				(query.state.data?.data ?? []).some((recording) => isProcessing(recording.status))
					? PROCESSING_POLL_MS
					: false
		}
	}));

	const conversationsQuery = createGetConversations(() => ({
		query: {
			refetchInterval: () =>
				(recordingsQuery.data?.data ?? []).some((recording) => isProcessing(recording.status))
					? PROCESSING_POLL_MS
					: false
		}
	}));

	const conversations = $derived(
		conversationsQuery.data?.status === 200 ? conversationsQuery.data.data : []
	);

	const recordings = $derived(
		recordingsQuery.data?.status === 200 ? (recordingsQuery.data.data ?? []) : []
	);

	const processing = $derived(recordings.filter((recording) => isProcessing(recording.status)));

	// Completed conversations are listed above; this surfaces anything that will
	// never appear there (still processing or failed).
	const pending = $derived(
		recordings.filter(
			(recording) =>
				isProcessing(recording.status) ||
				(recording.status === 'failed' &&
					!conversations.some((conversation) => conversation.recording_id === recording.id))
		)
	);

	let uploadOpen = $state(false);
</script>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-6 p-6 lg:p-8">
	<div class="flex flex-wrap items-end justify-between gap-4">
		<div class="flex flex-col gap-1">
			<h1 class="text-2xl font-semibold tracking-tight">Conversations</h1>
			<p class="text-sm text-muted-foreground">
				{#if conversationsQuery.isPending}
					Loading conversations…
				{:else}
					{conversations.length}
					{conversations.length === 1 ? 'conversation' : 'conversations'} transcribed
					{#if processing.length > 0}
						· {processing.length} processing
					{/if}
				{/if}
			</p>
		</div>
		<Button onclick={() => (uploadOpen = true)}>
			<UploadIcon />
			Upload recording
		</Button>
	</div>

	{#if pending.length > 0}
		<Card.Root class="gap-0 py-0">
			<Card.Header class="border-b py-3">
				<Card.Title class="flex items-center gap-2 text-sm font-medium">
					{#if processing.length > 0}
						<Loader2Icon class="size-4 animate-spin text-muted-foreground" />
						Processing {processing.length} recording{processing.length === 1 ? '' : 's'}
					{:else}
						Unfinished recordings
					{/if}
				</Card.Title>
				<Card.Description>
					You can keep using the platform while these finish — this list updates automatically.
				</Card.Description>
			</Card.Header>
			<div class="flex flex-col divide-y">
				{#each pending as recording (recording.id)}
					<div class="flex flex-wrap items-center gap-3 px-4 py-3">
						<div class="flex min-w-0 flex-1 flex-col">
							<span class="truncate text-sm font-medium">{recording.filename || 'Untitled'}</span>
							<span class="truncate text-xs text-muted-foreground">
								{recording.extension?.toUpperCase() ?? 'Audio'}
							</span>
						</div>
						<StatusBadge status={recording.status} />
					</div>
				{/each}
			</div>
		</Card.Root>
	{/if}

	{#if conversationsQuery.isError}
		<Card.Root>
			<Card.Content class="text-sm text-destructive">
				Failed to load conversations. Make sure the API is running and try again.
			</Card.Content>
		</Card.Root>
	{:else if conversationsQuery.isPending}
		<Card.Root class="gap-0 py-0">
			<div class="flex flex-col divide-y">
				{#each Array.from({ length: 5 }, (_, i) => i) as i (i)}
					<div class="flex items-center gap-4 p-4">
						<Skeleton class="size-9 rounded-full" />
						<div class="flex flex-1 flex-col gap-2">
							<Skeleton class="h-4 w-1/3" />
							<Skeleton class="h-3 w-1/4" />
						</div>
						<Skeleton class="h-5 w-20 rounded-full" />
					</div>
				{/each}
			</div>
		</Card.Root>
	{:else if conversations.length === 0}
		<Card.Root>
			<Card.Content class="flex flex-col items-center justify-center gap-3 py-16 text-center">
				<div class="flex size-12 items-center justify-center rounded-full bg-muted">
					<MessagesSquareIcon class="size-6 text-muted-foreground" />
				</div>
				<div class="flex flex-col gap-1">
					<p class="font-medium">No conversations yet</p>
					<p class="text-sm text-muted-foreground">
						Upload a recording to get your first transcript.
					</p>
				</div>
				<Button class="mt-2" onclick={() => (uploadOpen = true)}>
					<UploadIcon />
					Upload recording
				</Button>
			</Card.Content>
		</Card.Root>
	{:else}
		<Card.Root class="gap-0 py-0">
			<Table.Root>
				<Table.Header>
					<Table.Row class="hover:bg-transparent">
						<Table.Head>Title</Table.Head>
						<Table.Head>Meeting date</Table.Head>
						<Table.Head>Duration</Table.Head>
						<Table.Head>Language</Table.Head>
						<Table.Head>Speakers</Table.Head>
						<Table.Head class="text-right">Status</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each conversations as conversation (conversation.id)}
						<Table.Row>
							<Table.Cell class="max-w-xs">
								<a
									href={resolve('/conversations/[id]', { id: conversation.id })}
									class="font-medium underline-offset-4 hover:underline"
								>
									{conversation.title || 'Untitled conversation'}
								</a>
								{#if conversation.summary}
									<p class="mt-0.5 line-clamp-1 max-w-sm text-xs text-muted-foreground">
										{conversation.summary.replace(/[#*_`>]/g, '').trim()}
									</p>
								{/if}
							</Table.Cell>
							<Table.Cell class="text-xs whitespace-nowrap text-muted-foreground">
								{@const date = conversationDate(conversation.meeting_at, conversation.created_at)}
								<span title={date.isUploaded ? 'Uploaded' : 'Meeting date'}>
									{date.label}{#if date.isUploaded}<span class="text-muted-foreground/70">
											(uploaded)</span
										>{/if}
								</span>
							</Table.Cell>
							<Table.Cell class="whitespace-nowrap">
								<span class="inline-flex items-center gap-1 text-xs text-muted-foreground">
									<ClockIcon class="size-3" />
									{formatDuration(conversation.length)}
								</span>
							</Table.Cell>
							<Table.Cell class="text-xs text-muted-foreground uppercase">
								{conversation.language || '—'}
							</Table.Cell>
							<Table.Cell>
								<span class="inline-flex items-center gap-1 text-sm">
									{conversation.speaker_count}
									{#if conversation.unknown_speaker_count > 0}
										<TriangleAlertIcon
											class="size-3.5 text-amber-600 dark:text-amber-400"
											aria-label="{conversation.unknown_speaker_count} unknown speakers"
										/>
									{/if}
								</span>
							</Table.Cell>
							<Table.Cell class="text-right">
								<StatusBadge status={conversation.recording_status} />
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</Card.Root>
	{/if}
</div>

<UploadRecordingDialog bind:open={uploadOpen} />
