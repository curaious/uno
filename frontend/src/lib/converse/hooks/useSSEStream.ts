export interface SSEStreamOptions {
  onChunk: (data: string) => void;
  onError?: (error: Error) => void;
  onComplete?: () => void;
}

/**
 * Streams SSE (Server-Sent Events) from a URL.
 * Parses SSE frames and calls onChunk for each data payload.
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
