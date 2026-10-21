<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { formatTimestamp } from '#lib/api-helpers.js';
	import { cn } from '#lib/utils.js';
	import PlayIcon from '@lucide/svelte/icons/play';
	import PauseIcon from '@lucide/svelte/icons/pause';

	let {
		src = null,
		class: className,
		onRangeChange
	}: {
		src: string | null;
		class?: string;
		/** Called when a segment-range playback starts (`range`) or ends (`null`). */
		onRangeChange?: (range: { start: number; end: number } | null) => void;
	} = $props();

	let audio = $state<HTMLAudioElement | null>(null);
	let currentTime = $state(0);
	let duration = $state(0);
	let paused = $state(true);
	let rangeEnd = $state<number | null>(null);
	let rate = $state('1');

	const progressPercent = $derived(duration > 0 ? (currentTime / duration) * 100 : 0);

	/** Start playing the segment between `start` and `end` seconds. */
	export function playRange(start: number, end: number) {
		if (!audio) return;
		rangeEnd = end;
		onRangeChange?.({ start, end });
		audio.currentTime = Math.max(0, start);
		void audio.play().catch(() => {
			// Autoplay can be blocked by the browser; leave the element paused.
		});
	}

	/** Pause playback and clear the active segment. */
	export function stop() {
		audio?.pause();
		rangeEnd = null;
		onRangeChange?.(null);
	}

	export function seek(time: number) {
		if (!audio) return;
		audio.currentTime = Math.max(0, Math.min(time, duration || time));
	}

	function togglePlay() {
		if (!audio) return;
		if (audio.paused) {
			rangeEnd = null;
			onRangeChange?.(null);
			void audio.play().catch(() => {});
		} else {
			audio.pause();
		}
	}

	function onTimeUpdate() {
		if (!audio) return;
		currentTime = audio.currentTime;
		// Auto-pause at the end of a segment range.
		const end = rangeEnd;
		if (end !== null && audio.currentTime >= end) {
			audio.pause();
			rangeEnd = null;
			onRangeChange?.(null);
		}
	}

	function onLoadedMetadata() {
		const value = audio?.duration;
		duration = value != null && Number.isFinite(value) ? value : 0;
	}

	function onSeek(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		rangeEnd = null;
		onRangeChange?.(null);
		seek(Number(input.value));
	}
</script>

<div
	class={cn(
		'flex flex-col gap-3 rounded-xl bg-card p-4 text-card-foreground shadow-xs ring-1 ring-foreground/10',
		className
	)}
>
	<audio
		bind:this={audio}
		{src}
		preload="metadata"
		onplay={() => (paused = false)}
		onpause={() => (paused = true)}
		onended={() => {
			paused = true;
			rangeEnd = null;
			onRangeChange?.(null);
		}}
		ontimeupdate={onTimeUpdate}
		onloadedmetadata={onLoadedMetadata}
		ondurationchange={onLoadedMetadata}
	></audio>

	{#if !src}
		<p class="text-sm text-muted-foreground">No audio available for this recording.</p>
	{:else}
		<div class="flex items-center gap-3">
			<Button
				variant="default"
				size="icon"
				class="shrink-0 rounded-full"
				onclick={togglePlay}
				aria-label={paused ? 'Play' : 'Pause'}
			>
				{#if paused}
					<PlayIcon class="translate-x-px" />
				{:else}
					<PauseIcon />
				{/if}
			</Button>

			<div class="flex flex-1 items-center gap-3">
				<span class="w-12 shrink-0 text-right font-mono text-xs tabular-nums">
					{formatTimestamp(currentTime)}
				</span>

				<input
					type="range"
					min="0"
					max={duration || 0}
					step="0.1"
					value={currentTime}
					oninput={onSeek}
					aria-label="Seek"
					class="h-1.5 w-full cursor-pointer appearance-none rounded-full outline-none [&::-moz-range-thumb]:size-3 [&::-moz-range-thumb]:rounded-full [&::-moz-range-thumb]:border-0 [&::-moz-range-thumb]:bg-primary [&::-webkit-slider-thumb]:size-3 [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-primary [&::-webkit-slider-thumb]:shadow"
					style:--progress={progressPercent}
					disabled={duration === 0}
				/>

				<span class="w-12 shrink-0 font-mono text-xs text-muted-foreground tabular-nums">
					{formatTimestamp(duration)}
				</span>
			</div>

			<Select.Root
				type="single"
				value={rate}
				onValueChange={(value) => {
					if (value) {
						rate = value;
						if (audio) audio.playbackRate = Number(value);
					}
				}}
			>
				<Select.Trigger size="sm" class="w-20 shrink-0" aria-label="Playback speed">
					{rate}×
				</Select.Trigger>
				<Select.Content>
					{#each ['0.5', '0.75', '1', '1.25', '1.5', '1.75', '2'] as speed (speed)}
						<Select.Item value={speed} label="{speed}×" />
					{/each}
				</Select.Content>
			</Select.Root>
		</div>

		{#if rangeEnd !== null}
			<p class="text-xs text-muted-foreground">
				Playing segment · stops at {formatTimestamp(rangeEnd)}
			</p>
		{/if}
	{/if}
</div>

<style>
	input[type='range'] {
		background: linear-gradient(
			to right,
			var(--primary) 0%,
			var(--primary) var(--progress),
			var(--muted) var(--progress),
			var(--muted) 100%
		);
	}
</style>
