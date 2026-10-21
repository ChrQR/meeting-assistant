<script lang="ts">
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';
	import { uploadRecording } from '../../api.js';
	import { errorMessage, invalidateByPrefix } from '#lib/api-helpers.js';
	import UploadIcon from '@lucide/svelte/icons/upload';
	import Loader2Icon from '@lucide/svelte/icons/loader-2';

	let {
		open = $bindable(false),
		onOpenChange = () => {}
	}: { open?: boolean; onOpenChange?: (open: boolean) => void } = $props();

	const queryClient = useQueryClient();

	let title = $state('');
	let file = $state<File | null>(null);
	let fileInput = $state<HTMLInputElement | null>(null);
	let meetingDate = $state(todayISO());

	const uploadMutation = createMutation(() => ({
		mutationFn: (vars: { title: string; file: File; meeting_at?: string }) => uploadRecording(vars),
		onSuccess: async () => {
			toast.success('Recording uploaded', {
				description: 'Transcription is running in the background — you can keep using the app.'
			});
			await invalidateByPrefix(queryClient, '/recordings', '/conversations');
			handleOpenChange(false);
		},
		onError: (error: unknown) => {
			toast.error('Upload failed', { description: errorMessage(error) });
		}
	}));

	function todayISO() {
		const now = new Date();
		const offset = now.getTimezoneOffset() * 60_000;
		return new Date(now.getTime() - offset).toISOString().slice(0, 10);
	}

	function reset() {
		title = '';
		file = null;
		meetingDate = todayISO();
		if (fileInput) fileInput.value = '';
	}

	/** Use with `{@attach}` so the state resets whenever the dialog persists into the DOM. */
	function resetOnMount() {
		reset();
	}

	function defaultTitleFor(name: string) {
		return name
			.replace(/\.[^.]+$/, '')
			.replaceAll(/[_-]+/g, ' ')
			.trim();
	}

	function handleOpenChange(value: boolean) {
		open = value;
		onOpenChange(value);
		if (!value) reset();
	}

	function onFileChange(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		file = input.files?.[0] ?? null;
		if (file && !title.trim()) title = defaultTitleFor(file.name);
	}

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		if (!file || !title.trim() || uploadMutation.isPending) return;
		uploadMutation.mutate({
			title: title.trim(),
			file,
			meeting_at: meetingDate || undefined
		});
	}
</script>

<Dialog.Root bind:open onOpenChange={handleOpenChange}>
	<Dialog.Content>
		<form onsubmit={handleSubmit} {@attach resetOnMount} class="flex flex-col gap-6">
			<Dialog.Header>
				<Dialog.Title>Upload recording</Dialog.Title>
				<Dialog.Description>
					Upload an audio file to transcribe. Processing happens in the background.
				</Dialog.Description>
			</Dialog.Header>

			<div class="flex flex-col gap-4">
				<div class="flex flex-col gap-2">
					<Label for="upload-title">Title</Label>
					<Input
						id="upload-title"
						bind:value={title}
						placeholder="Weekly team sync"
						required
						disabled={uploadMutation.isPending}
					/>
				</div>

				<div class="flex flex-col gap-2">
					<Label for="upload-date">Meeting date</Label>
					<Input
						id="upload-date"
						type="date"
						bind:value={meetingDate}
						disabled={uploadMutation.isPending}
					/>
					<p class="text-xs text-muted-foreground">
						When the meeting took place. Defaults to today.
					</p>
				</div>

				<div class="flex flex-col gap-2">
					<Label for="upload-file">Recording file</Label>
					<input
						bind:this={fileInput}
						id="upload-file"
						type="file"
						accept="audio/*,video/mp4,video/x-m4v,.m4a,.mp4,.mov"
						required
						onchange={onFileChange}
						disabled={uploadMutation.isPending}
						class="w-full cursor-pointer rounded-md border border-dashed border-input p-2 text-sm text-muted-foreground file:mr-3 file:cursor-pointer file:rounded-sm file:border-0 file:bg-muted file:px-2.5 file:py-1 file:text-xs file:font-medium file:text-foreground hover:file:bg-muted/80"
					/>
					<p class="text-xs text-muted-foreground">
						Audio or video files (MP3, M4A, WAV, MP4/MOV from Teams, Zoom or Meet). Only the audio
						track is used.
					</p>
					{#if file}
						<p class="text-xs text-muted-foreground">
							{file.name} · {(file.size / 1024 / 1024).toFixed(2)} MB
						</p>
					{/if}
				</div>
			</div>

			<Dialog.Footer>
				<Button
					type="button"
					variant="outline"
					onclick={() => handleOpenChange(false)}
					disabled={uploadMutation.isPending}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={!file || !title.trim() || uploadMutation.isPending}>
					{#if uploadMutation.isPending}
						<Loader2Icon class="animate-spin" />
						Uploading…
					{:else}
						<UploadIcon />
						Upload
					{/if}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
