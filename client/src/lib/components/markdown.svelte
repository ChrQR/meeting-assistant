<script lang="ts">
	import { marked } from 'marked';
	import DOMPurify from 'dompurify';

	let { content }: { content: string } = $props();

	// Summaries are model output, so always sanitize before rendering as HTML.
	const html = $derived(DOMPurify.sanitize(marked.parse(content, { async: false }) as string));
</script>

<div
	class="text-sm leading-relaxed [&_a]:underline [&_a]:underline-offset-4 [&_code]:rounded [&_code]:bg-muted [&_code]:px-1 [&_code]:py-0.5 [&_h1]:mt-4 [&_h1]:mb-2 [&_h1]:text-base [&_h1]:font-semibold [&_h2]:mt-4 [&_h2]:mb-2 [&_h2]:text-sm [&_h2]:font-semibold [&_h3]:mt-3 [&_h3]:mb-1 [&_h3]:font-semibold [&_li]:ml-5 [&_li]:list-disc [&_ol_li]:list-decimal [&_p]:mb-2 [&_strong]:font-semibold [&>*:first-child]:mt-0 [&>*:last-child]:mb-0"
>
	<!-- eslint-disable-next-line svelte/no-at-html-tags -->
	{@html html}
</div>
