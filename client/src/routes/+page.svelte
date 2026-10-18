<script lang="ts">
	import { createGetRecordings } from '../api';

	const recordingsQuery = createGetRecordings();
</script>

<div class="h-screen w-screen items-center justify-center">
	{#if recordingsQuery.isPending}
		<p>Loading recordings...</p>
	{:else if recordingsQuery.isError}
		<p>Failed to load recordings.</p>
	{:else if recordingsQuery.data?.status === 200}
		<ul>
			{#each recordingsQuery.data.data as recording}
				<li>
					<span>ID: {recording.id}</span>
					<span>Ext: {recording.extension}</span>
					<span>File: {recording.filename || '(No filename)'}</span>
					<span>Status: {recording.status}</span>
				</li>
			{/each}
		</ul>
	{/if}
</div>
