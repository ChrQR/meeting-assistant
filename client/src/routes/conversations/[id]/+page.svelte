<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import type { ConversationSegment, Speaker } from '../../../api.js';
	import { createGetConversation, setSegmentIgnored } from '../../../api.js';
	import {
		conversationDate,
		formatDuration,
		invalidateByPrefix,
		recordingAudioUrl
	} from '#lib/api-helpers.js';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import AudioPlayer from '#lib/components/audio-player.svelte';
	import AssignSpeakerDialog from '#lib/components/assign-speaker-dialog.svelte';
	import TranscriptSegment from '#lib/components/transcript-segment.svelte';
	import Markdown from '#lib/components/markdown.svelte';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';
	import ClockIcon from '@lucide/svelte/icons/clock';
	import UsersIcon from '@lucide/svelte/icons/users';
	import GlobeIcon from '@lucide/svelte/icons/globe';
	import SparklesIcon from '@lucide/svelte/icons/sparkles';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import UserRoundPlusIcon from '@lucide/svelte/icons/user-round-plus';
	import EyeOffIcon from '@lucide/svelte/icons/eye-off';
	import EyeIcon from '@lucide/svelte/icons/eye';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';
	import { errorMessage } from '#lib/api-helpers.js';

	const id = $derived(page.params.id ?? '');
	const conversationQuery = createGetConversation(() => id);

	const conversation = $derived(
		conversationQuery.data?.status === 200 ? conversationQuery.data.data : null
	);
	const segments: ConversationSegment[] = $derived(conversation?.segments ?? []);
	const audioSrc = $derived(recordingAudioUrl(conversation?.recording_id));

	let showIgnored = $state(false);
	const ignoredCount = $derived(segments.filter((segment) => segment.ignored).length);
	const visibleSegments = $derived(
		showIgnored ? segments : segments.filter((segment) => !segment.ignored)
	);
	const secondaryDate = $derived(
		conversationDate(conversation?.meeting_at, conversation?.created_at)
	);

	// A distinct speaker id may appear in several segments; treat the speaker as
	// unknown only when none of its segments carry a name. Ignored segments are
	// dismissed and must not keep the warning alive.
	const unknownSegments = $derived(
		segments.filter(
			(segment) =>
				!segment.ignored &&
				segment.speaker_name === null &&
				!segments.some((other) => other.speaker_id === segment.speaker_id && other.speaker_name)
		)
	);

	const unknownSpeakerIds = $derived([
		...new Set(unknownSegments.map((segment) => segment.speaker_id))
	]);

	let player = $state<ReturnType<typeof AudioPlayer> | null>(null);
	let activeSegmentId = $state<string | null>(null);
	let assignOpen = $state(false);
	const queryClient = useQueryClient();

	const ignoreMutation = createMutation(() => ({
		mutationFn: (input: { segmentId: string; ignored: boolean }) =>
			setSegmentIgnored(id, input.segmentId, { ignored: input.ignored }),
		onSuccess: async (_result, input) => {
			toast.success(input.ignored ? 'Segment ignored' : 'Segment restored', {
				description: input.ignored ? 'It stays in the transcript behind "Show ignored".' : undefined
			});
			await invalidateByPrefix(queryClient, '/conversations', '/speakers');
		},
		onError: (error: unknown) => {
			toast.error('Could not update segment', { description: errorMessage(error) });
		}
	}));

	function toggleIgnored(segment: ConversationSegment) {
		ignoreMutation.mutate({ segmentId: segment.id, ignored: !segment.ignored });
	}
	let assignTargetId = $state<string | null>(null);

	const assignTarget = $derived(
		unknownSegments.find((segment) => segment.speaker_id === assignTargetId) ?? null
	);

	/** Minimal `Speaker`-shaped object for the dialog (we only have segment data here). */
	const assignSpeaker: Speaker | null = $derived(
		assignTarget
			? {
					id: assignTarget.speaker_id,
					name: null,
					email: null,
					company_id: null,
					company_name: null,
					user_id: null,
					user_email: null,
					embedding_count: 0,
					segment_count: segments.filter((s) => s.speaker_id === assignTarget.speaker_id).length,
					total_speaking_time: segments
						.filter((s) => s.speaker_id === assignTarget.speaker_id)
						.reduce((total, s) => total + s.duration, 0),
					conversation_count: 1,
					created_at: conversation?.created_at ?? new Date().toISOString()
				}
			: null
	);

	function playSegment(segment: ConversationSegment) {
		activeSegmentId = segment.id;
		player?.playRange(segment.start_time, segment.end_time);
	}

	function openAssign(speakerId: string) {
		assignTargetId = speakerId;
		assignOpen = true;
	}
</script>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-6 p-6 lg:p-8">
	<a
		href={resolve('/')}
		class="inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
	>
		<ArrowLeftIcon class="size-4" />
		All conversations
	</a>

	{#if conversationQuery.isPending}
		<div class="flex flex-col gap-3">
			<Skeleton class="h-8 w-2/3" />
			<Skeleton class="h-4 w-1/3" />
		</div>
		<Skeleton class="h-24 w-full rounded-xl" />
		<Skeleton class="h-16 w-full rounded-xl" />
		<Skeleton class="h-96 w-full rounded-xl" />
	{:else if conversationQuery.isError || !conversation}
		<Card.Root>
			<Card.Content class="text-sm text-destructive">
				Conversation not found or could not be loaded.
			</Card.Content>
		</Card.Root>
	{:else}
		<div class="flex flex-wrap items-start justify-between gap-4">
			<div class="flex min-w-0 flex-col gap-2">
				<h1 class="text-2xl font-semibold tracking-tight">
					{conversation.title || 'Untitled conversation'}
				</h1>
				<div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
					<span>
						{secondaryDate.label}
						{#if secondaryDate.isUploaded}
							<span class="text-muted-foreground/70">(uploaded)</span>
						{/if}
					</span>
					<span class="inline-flex items-center gap-1">
						<ClockIcon class="size-3.5" />
						{formatDuration(conversation.length)}
					</span>
					{#if conversation.language}
						<span class="inline-flex items-center gap-1 uppercase">
							<GlobeIcon class="size-3.5" />
							{conversation.language}
						</span>
					{/if}
					<span class="inline-flex items-center gap-1">
						<UsersIcon class="size-3.5" />
						{conversation.speaker_count}
						{conversation.speaker_count === 1 ? 'speaker' : 'speakers'}
					</span>
				</div>
			</div>
			<StatusBadge status={conversation.recording_status} />
		</div>

		{#if unknownSpeakerIds.length > 0}
			<div
				class="flex flex-wrap items-center gap-3 rounded-lg border border-amber-500/30 bg-amber-500/5 px-4 py-3 text-sm"
			>
				<TriangleAlertIcon class="size-4 shrink-0 text-amber-600 dark:text-amber-400" />
				<span class="flex-1">
					{unknownSpeakerIds.length}
					{unknownSpeakerIds.length === 1 ? 'speaker is' : 'speakers are'} not identified yet.
				</span>
				<Button size="sm" variant="outline" onclick={() => openAssign(unknownSpeakerIds[0])}>
					<UserRoundPlusIcon />
					Assign
				</Button>
			</div>
		{/if}

		<Card.Root>
			<Card.Header>
				<Card.Title class="flex items-center gap-2">
					<SparklesIcon class="size-4" />
					Summary
				</Card.Title>
			</Card.Header>
			<Card.Content>
				{#if conversation.summary}
					<Markdown content={conversation.summary} />
				{:else}
					<p class="text-sm text-muted-foreground">No summary yet.</p>
				{/if}
			</Card.Content>
		</Card.Root>

		<AudioPlayer
			bind:this={player}
			src={audioSrc}
			onRangeChange={(range) => {
				if (range === null) activeSegmentId = null;
			}}
		/>

		<Card.Root class="gap-0 py-0">
			<Card.Header class="gap-1.5 border-b py-4">
				<Card.Title>Transcript</Card.Title>
				<Card.Description>
					{#if segments.length > 0}
						{visibleSegments.length} of {segments.length} segments · click a segment to jump to that part
						of the audio
					{:else}
						No transcript yet.
					{/if}
				</Card.Description>
				{#if ignoredCount > 0}
					<Card.Action>
						<Button variant="ghost" size="sm" onclick={() => (showIgnored = !showIgnored)}>
							{#if showIgnored}
								<EyeOffIcon />
								Hide ignored
							{:else}
								<EyeIcon />
								Show {ignoredCount} ignored
							{/if}
						</Button>
					</Card.Action>
				{/if}
			</Card.Header>
			<div class="max-h-[65vh] overflow-y-auto p-2">
				{#if segments.length === 0}
					<p class="p-6 text-center text-sm text-muted-foreground">
						The transcript will appear here once the recording has been transcribed.
					</p>
				{:else}
					<div class="flex flex-col gap-0.5">
						{#each visibleSegments as segment (segment.id)}
							<TranscriptSegment
								{segment}
								active={activeSegmentId === segment.id}
								onPlay={playSegment}
								onAssign={openAssign}
								onToggleIgnored={toggleIgnored}
							/>
						{/each}
					</div>
				{/if}
			</div>
		</Card.Root>

		<Separator />
		<p class="text-xs text-muted-foreground">
			Conversation ID <span class="font-mono">{conversation.id}</span>
		</p>
	{/if}
</div>

<AssignSpeakerDialog
	speaker={assignSpeaker}
	bind:open={assignOpen}
	onOpenChange={(open) => {
		if (!open) assignTargetId = null;
	}}
/>
