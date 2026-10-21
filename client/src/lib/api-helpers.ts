import type { QueryClient } from '@tanstack/svelte-query';

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';

/**
 * Absolute URL of a recording's audio stream. Used directly as the <audio>
 * source so the browser can issue range requests against the backend.
 */
export function recordingAudioUrl(recordingId: string | null | undefined): string | null {
	if (!recordingId) return null;
	return `${BASE_URL}/recordings/${recordingId}/audio`;
}

/** Formats a duration in seconds as `m:ss` (or `h:mm:ss` when needed). */
export function formatTimestamp(seconds: number | null | undefined): string {
	if (seconds == null || !Number.isFinite(seconds)) return '0:00';
	const total = Math.max(0, Math.floor(seconds));
	const hours = Math.floor(total / 3600);
	const minutes = Math.floor((total % 3600) / 60);
	const secs = total % 60;
	if (hours > 0) {
		return `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
	}
	return `${minutes}:${String(secs).padStart(2, '0')}`;
}

/** Human readable duration, e.g. `4m 12s`. */
export function formatDuration(seconds: number | null | undefined): string {
	if (seconds == null || !Number.isFinite(seconds) || seconds <= 0) return '—';
	const total = Math.round(seconds);
	if (total < 60) return `${total}s`;
	const minutes = Math.floor(total / 60);
	const secs = total % 60;
	if (minutes < 60) return secs > 0 ? `${minutes}m ${secs}s` : `${minutes}m`;
	const hours = Math.floor(minutes / 60);
	const rest = minutes % 60;
	return rest > 0 ? `${hours}h ${rest}m` : `${hours}h`;
}

/**
 * Display date for a conversation: the meeting date when known, otherwise the
 * upload time. Returns the label and whether it is a fallback.
 */
export function conversationDate(
	meetingAt: string | null | undefined,
	createdAt: string | null | undefined
): { label: string; isUploaded: boolean } {
	if (meetingAt) return { label: formatDate(meetingAt), isUploaded: false };
	return { label: formatDate(createdAt), isUploaded: true };
}

export function formatDate(value: string | null | undefined): string {
	if (!value) return '—';
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return '—';
	return date.toLocaleString(undefined, {
		day: 'numeric',
		month: 'short',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit'
	});
}

const SPEAKER_COLORS = [
	'bg-blue-500',
	'bg-emerald-500',
	'bg-amber-500',
	'bg-violet-500',
	'bg-rose-500',
	'bg-cyan-600',
	'bg-orange-500',
	'bg-indigo-500'
];

/** Deterministic color class for a speaker id, so a speaker is always the same color. */
export function speakerColor(id: string): string {
	let hash = 0;
	for (let i = 0; i < id.length; i++) {
		hash = (hash * 31 + id.charCodeAt(i)) | 0;
	}
	return SPEAKER_COLORS[Math.abs(hash) % SPEAKER_COLORS.length];
}

export function initials(name: string | null | undefined): string {
	if (!name) return '?';
	const parts = name.trim().split(/\s+/).slice(0, 2);
	return parts.map((part) => part[0]?.toUpperCase() ?? '').join('') || '?';
}

/** Recording statuses that mean the pipeline is still working. */
export const IN_PROGRESS_STATUSES = [
	'pending',
	'uploaded',
	'transcribing',
	'transcribed',
	'embedding',
	'embedded',
	'assigning_speakers',
	'summarizing',
	'summarized'
] as const;

export function isProcessing(status: string | null | undefined): boolean {
	return status != null && (IN_PROGRESS_STATUSES as readonly string[]).includes(status);
}

/** Extracts a human readable message from an unknown thrown value. */
export function errorMessage(error: unknown, fallback = 'Something went wrong'): string {
	if (error instanceof Error && error.message) return error.message;
	if (typeof error === 'string' && error) return error;
	return fallback;
}

/**
 * Orval generates query keys as single-element arrays containing the URL
 * (`['/speakers/:id']`), so prefix-based array matching does not reach detail
 * queries. This predicate matches every query whose key starts with one of the
 * given URL prefixes.
 */
export function invalidateByPrefix(queryClient: QueryClient, ...prefixes: string[]): Promise<void> {
	return queryClient.invalidateQueries({
		predicate: (query) => {
			const [key] = query.queryKey;
			return typeof key === 'string' && prefixes.some((prefix) => key.startsWith(prefix));
		}
	});
}
