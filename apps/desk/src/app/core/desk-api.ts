import { Service } from '@angular/core';
import { Guide, Meta, PromptInfo, TurnRequest, TurnResponse } from './types';

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

  turn(req: TurnRequest): Promise<TurnResponse> {
    return this.request<TurnResponse>('POST', '/api/turn', req);
  }

  private async request<T>(method: 'GET' | 'POST', path: string, body?: unknown): Promise<T> {
    const res = await fetch(path, {
      method,
      cache: 'no-store',
      headers: { 'X-SDD-Token': this.token, ...(body === undefined ? {} : { 'Content-Type': 'application/json' }) },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    const data = (await res.json().catch(() => ({}))) as { error?: string };
    if (!res.ok) {
      throw new ApiError(data.error ?? res.statusText, res.status);
    }
    return data as T;
  }
}
