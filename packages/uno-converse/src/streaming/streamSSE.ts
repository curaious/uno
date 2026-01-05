/**
 * Options for the SSE stream callbacks
 */
export interface SSEStreamOptions {
  /** Called for each data chunk received */
  onChunk: (data: string) => void;
  /** Called when an error occurs */
  onError?: (error: Error) => void;
  /** Called when the stream completes */
  onComplete?: () => void;
}

/**
 * Streams Server-Sent Events (SSE) from a URL.
 * Parses SSE frames and calls onChunk for each data payload.
 *
 * @param url - The URL to stream from
 * @param requestOptions - Fetch request options
 * @param callbacks - SSE event callbacks
 * @param abortSignal - Optional signal to abort the stream
 *
 * @example
 * ```ts
 * await streamSSE(
 *   'https://api.example.com/stream',
 *   { method: 'POST', body: JSON.stringify({ message: 'Hello' }) },
 *   {
 *     onChunk: (data) => console.log('Received:', data),
 *     onComplete: () => console.log('Done'),
 *     onError: (err) => console.error('Error:', err),
 *   }
 * );
 * ```
 */
export async function streamSSE(
  url: string,
  requestOptions: RequestInit,
  callbacks: SSEStreamOptions,
  abortSignal?: AbortSignal
): Promise<void> {
  try {
    const response = await fetch(url, {
      ...requestOptions,
      signal: abortSignal,
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    if (!response.body) {
      throw new Error('Response body is null');
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // Parse SSE frames: split on double newline
      let idx;
      while ((idx = buffer.indexOf('\n\n')) !== -1) {
        const frame = buffer.slice(0, idx);
        buffer = buffer.slice(idx + 2);

        // Join all data: lines in the frame
        const data = frame
          .split('\n')
          .filter(line => line.startsWith('data:'))
          .map(line => line.slice(5).trim())
          .join('\n');

        if (data) {
          callbacks.onChunk(data);
        }
      }
    }

    // Final flush of decoder state
    decoder.decode();
    callbacks.onComplete?.();
  } catch (error) {
    if ((error as Error).name === 'AbortError') {
      return;
    }
    callbacks.onError?.(error as Error);
  }
}

