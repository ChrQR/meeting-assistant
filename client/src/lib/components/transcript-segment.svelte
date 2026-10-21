<script lang="ts">
	import type { ConversationSegment } from '../../api.js';
	import { formatTimestamp, initials, speakerColor } from '#lib/api-helpers.js';
	import { cn } from '#lib/utils.js';
	import PlayIcon from '@lucide/svelte/icons/play';
	import PauseIcon from '@lucide/svelte/icons/pause';
	import UserRoundPlusIcon from '@lucide/svelte/icons/user-round-plus';
	import EyeOffIcon from '@lucide/svelte/icons/eye-off';
	import EyeIcon from '@lucide/svelte/icons/eye';

	let {
		segment,
		active = false,
		onPlay,
		onAssign,
		onToggleIgnored
	}: {
		segment: ConversationSegment;
		active?: boolean;
		onPlay?: (segment: ConversationSegment) => void;
		onAssign?: (speakerId: string) => void;
		onToggleIgnored?: (segment: ConversationSegment) => void;
	} = $props();

	const isUnknown = $derived(segment.speaker_name === null);
	const color = $derived(speakerColor(segment.speaker_id));

	function play() {
		onPlay?.(segment);
	}

	function assign(event: MouseEvent) {
		event.stopPropagation();
		onAssign?.(segment.speaker_id);
	}

	function toggleIgnored(event: MouseEvent) {
		event.stopPropagation();
		onToggleIgnored?.(segment);
	}

	// Ignored segments are dismissed noise; they are rendered muted and cannot be
	// assigned to a speaker.
	const ignored = $derived(segment.ignored);
</script>

<div
	class={cn(
		'group flex gap-3 rounded-lg px-3 py-2.5 transition-colors',
		ignored ? 'opacity-60' : '',
		active ? 'bg-primary/5 ring-1 ring-primary/20' : 'hover:bg-muted/60'
	)}
>
	<button
		type="button"
		class="flex min-w-0 flex-1 cursor-pointer gap-3 text-left"
		onclick={play}
		aria-label="Play segment starting at {formatTimestamp(segment.start_time)}"
	>
		<span
			class={cn(
				'flex size-8 shrink-0 items-center justify-center rounded-full text-xs font-semibold text-white',
				isUnknown ? 'bg-amber-500' : color
			)}
			aria-hidden="true"
		>
			{initials(segment.speaker_name)}
		</span>

		<span class="flex min-w-0 flex-1 flex-col gap-1">
			<span class="flex flex-wrap items-center gap-2">
				{#if isUnknown}
					<span class="text-sm font-medium text-amber-700 dark:text-amber-400">
						Unknown speaker
					</span>
				{:else}
					<span class="text-sm font-medium">{segment.speaker_name}</span>
				{/if}

				<span class="font-mono text-xs text-muted-foreground tabular-nums">
					{formatTimestamp(segment.start_time)}
				</span>

				<span
					class={cn(
						'ml-auto flex size-6 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary transition-opacity',
						active ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'
					)}
				>
					{#if active}
						<PauseIcon class="size-3" />
					{:else}
						<PlayIcon class="size-3" />
					{/if}
				</span>
			</span>

			<span
				class={cn('text-sm leading-relaxed', active ? 'text-foreground' : 'text-foreground/90')}
			>
				{segment.text}
			</span>
		</span>
	</button>

	<div class="my-1 flex shrink-0 flex-col items-end gap-1 self-start">
		{#if isUnknown && onAssign && !ignored}
			<button
				type="button"
				class="inline-flex h-6 items-center gap-1 rounded-full bg-amber-500/10 px-2 text-xs font-medium text-amber-700 transition-colors hover:bg-amber-500/20 dark:text-amber-400"
				onclick={assign}
			>
				<UserRoundPlusIcon class="size-3" />
				Assign speaker
			</button>
		{/if}
		{#if isUnknown && onToggleIgnored}
			<button
				type="button"
				class="inline-flex h-6 items-center gap-1 rounded-full bg-muted px-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted/80 hover:text-foreground"
				onclick={toggleIgnored}
				title={ignored ? 'Restore this segment' : 'Ignore this segment'}
			>
				{#if ignored}
					<EyeIcon class="size-3" />
					Restore
				{:else}
					<EyeOffIcon class="size-3" />
					Ignore
				{/if}
			</button>
		{/if}
	</div>
</div>
