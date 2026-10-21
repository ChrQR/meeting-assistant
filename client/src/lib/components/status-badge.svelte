<script lang="ts">
	import { Badge, type BadgeVariant } from '#lib/components/ui/badge/index.js';
	import { cn } from '#lib/utils.js';
	import Loader2Icon from '@lucide/svelte/icons/loader-2';

	let { status, class: className }: { status: string | null | undefined; class?: string } =
		$props();

	type StatusMeta = {
		label: string;
		variant: BadgeVariant;
		class?: string;
		busy?: boolean;
	};

	const STATUS_MAP: Record<string, StatusMeta> = {
		completed: {
			label: 'Completed',
			variant: 'secondary',
			class: 'border-emerald-500/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
		},
		failed: { label: 'Failed', variant: 'destructive' },
		pending: { label: 'Pending', variant: 'outline' },
		uploaded: { label: 'Uploaded', variant: 'outline' },
		transcribing: { label: 'Transcribing', variant: 'secondary', busy: true },
		transcribed: { label: 'Transcribed', variant: 'secondary' },
		embedding: { label: 'Embedding', variant: 'secondary', busy: true },
		embedded: { label: 'Embedded', variant: 'secondary' },
		summarizing: { label: 'Summarizing', variant: 'secondary', busy: true },
		summarized: { label: 'Summarized', variant: 'secondary' },
		assigning_speakers: { label: 'Assigning speakers', variant: 'secondary', busy: true }
	};

	const meta = $derived<StatusMeta>(
		(status ? STATUS_MAP[status] : undefined) ?? {
			label: status ? status.replaceAll('_', ' ') : 'Unknown',
			variant: 'outline'
		}
	);

	const label = $derived(meta.label.charAt(0).toUpperCase() + meta.label.slice(1));
</script>

<Badge variant={meta.variant} class={cn('capitalize', meta.class, className)}>
	{#if meta.busy}
		<Loader2Icon class="animate-spin" />
	{/if}
	{label}
</Badge>
