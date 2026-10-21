<script lang="ts">
	import { page } from '$app/state';
	import { Button } from '#lib/components/ui/button/index.js';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
</script>

<div class="mx-auto flex w-full max-w-xl flex-col items-center gap-4 p-6 py-24 text-center lg:p-8">
	<div class="flex size-14 items-center justify-center rounded-full bg-muted">
		<TriangleAlertIcon class="size-7 text-muted-foreground" />
	</div>
	<p class="text-sm font-medium text-muted-foreground tabular-nums">{page.status}</p>
	<h1 class="text-2xl font-semibold tracking-tight">
		{page.status === 404 ? 'Page not found' : 'Something went wrong'}
	</h1>
	<p class="max-w-sm text-sm text-muted-foreground">
		{page.error?.message ?? 'An unexpected error occurred while loading this page.'}
	</p>
	<div class="flex gap-2">
		<Button variant="outline" onclick={() => history.back()}>Go back</Button>
		<Button href="/">
			<ArrowLeftIcon />
			Back to conversations
		</Button>
	</div>
</div>
