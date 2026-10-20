import { Service } from '@angular/core';
import { Guide, Meta, PromptInfo, StreamEvent, TurnRequest } from './types';

const TOKEN_KEY = 'sdd-desk-token';

/** Error of the sdd-desk API with its HTTP status (428 = disclosure not acknowledged). */
export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
  }
}

/**
 * Client of the local sdd-desk backend. The session token arrives once as ?token= in the URL, is kept in the tab's
 * session storage and removed from the address bar.
 */
@Service()
export class DeskApi {
  private readonly token = DeskApi.takeToken();

  private static takeToken(): string {
    const url = new URL(globalThis.location?.href ?? 'http://localhost/');
    const fromUrl = url.searchParams.get('token');
    if (fromUrl) {
      try {
        sessionStorage.setItem(TOKEN_KEY, fromUrl);
      } catch {
        /* storage blocked: keep it in memory only */
      }
      url.searchParams.delete('token');
      globalThis.history?.replaceState(null, '', url.pathname + url.hash);
      return fromUrl;
    }
    try {
      return sessionStorage.getItem(TOKEN_KEY) ?? '';
    } catch {
      return '';
    }
  }

  meta(): Promise<Meta> {
    return this.request<Meta>('GET', '/api/meta');
  }

  next(feature = ''): Promise<Guide> {
    return this.request<Guide>('GET', '/api/next' + (feature ? '?feature=' + encodeURIComponent(feature) : ''));
  }

  prompts(): Promise<{ prompts: PromptInfo[] }> {
    return this.request('GET', '/api/prompts');
  }

  prompt(name: string, args: Record<string, string>): Promise<{ messages: { content: { text: string } }[] }> {
    return this.request('POST', '/api/prompts/get', { name, arguments: args });
  }

  /**
   * Runs a conversation turn and calls onEvent for every event as it arrives. Rejects with ApiError if the backend
   * refuses the turn before streaming (e.g. 428), and with an AbortError when signal aborts.
   */
  async turnStream(req: TurnRequest, onEvent: (event: StreamEvent) => void, signal?: AbortSignal): Promise<void> {
    const res = await this.send('POST', '/api/turn/stream', req, signal);
    if (!res.ok || !res.body) {
      throw await apiError(res);
    }
    const lines = new NdjsonLines();
    const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
    for (;;) {
      const { value, done } = await reader.read();
      if (done) {
        break;
      }
      lines.push(value).forEach(onEvent);
    }
    lines.end().forEach(onEvent);
  }

  private async request<T>(method: 'GET' | 'POST', path: string, body?: unknown): Promise<T> {
    const res = await this.send(method, path, body);
    if (!res.ok) {
      throw await apiError(res);
    }
    return (await res.json().catch(() => ({}))) as T;
  }

  private send(method: 'GET' | 'POST', path: string, body?: unknown, signal?: AbortSignal): Promise<Response> {
    return fetch(path, {
      method,
      cache: 'no-store',
      headers: { 'X-SDD-Token': this.token, ...(body === undefined ? {} : { 'Content-Type': 'application/json' }) },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    });
  }
}

async function apiError(res: Response): Promise<ApiError> {
  const data = (await res.json().catch(() => ({}))) as { error?: string };
  return new ApiError(data.error ?? res.statusText, res.status);
}

/** Splits streamed text into NDJSON events; a line may arrive in several chunks. */
export class NdjsonLines {
  private buffer = '';

  push(chunk: string): StreamEvent[] {
    this.buffer += chunk;
    const parts = this.buffer.split('\n');
    this.buffer = parts.pop() ?? '';
    return parts.filter((l) => l.trim()).map((l) => JSON.parse(l) as StreamEvent);
  }

  end(): StreamEvent[] {
    const rest = this.buffer;
    this.buffer = '';
    return rest.trim() ? [JSON.parse(rest) as StreamEvent] : [];
  }
}
