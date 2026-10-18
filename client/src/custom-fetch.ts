const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';

export const customFetch = async <T>(
  url: string,
  options: RequestInit = {}
): Promise<T> => {
  const fullUrl = `${BASE_URL}${url}`;
  const res = await fetch(fullUrl, options);

  if (!res.ok) {
    const errorText = await res.text().catch(() => '');
    throw new Error(errorText || `Request failed with status ${res.status}`);
  }

  const body = [204, 205, 304].includes(res.status) ? null : await res.text();
  const parsedData = body ? JSON.parse(body) : undefined;

  return {
    data: parsedData,
    status: res.status,
    headers: res.headers,
  } as T;
};
