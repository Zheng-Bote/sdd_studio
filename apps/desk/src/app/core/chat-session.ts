import { computed, inject, Service, signal } from '@angular/core';
import { ApiError, DeskApi } from './desk-api';
import { Decision, Message, Meta, Pending, Provider } from './types';

/**
 * State of one conversation. The backend is stateless: every turn sends the conversation and the user's decisions.
 * Session permissions apply to read-only tools only; writes always need a one-time approval with preview.
 */
@Service()
export class ChatSession {
  private readonly api = inject(DeskApi);

  readonly meta = signal<Meta | null>(null);
  readonly providerId = signal('');
  readonly model = signal('');
  readonly messages = signal<Message[]>([]);
  readonly pending = signal<Pending[]>([]);
  readonly decisions = signal<Record<string, Decision>>({});
  readonly allowedTools = signal<string[]>([]);
  readonly acknowledged = signal<string[]>([]);
  readonly busy = signal(false);
  readonly error = signal('');
  /** Text of the answer the model is writing right now. */
  readonly streaming = signal('');
  /** The model's reasoning for the current step; shown while it works, never part of the conversation. */
  readonly thinking = signal('');
  readonly streamingMessage = computed<Message | null>(() =>
    this.streaming() ? { role: 'assistant', content: this.streaming() } : null,
  );

  readonly provider = computed<Provider | undefined>(() =>
    this.meta()?.providers.find((p) => p.id === this.providerId()),
  );
  /** A cloud provider must be acknowledged before the first call (ADR 0002). */
  readonly needsDisclosure = computed(() => {
    const p = this.provider();
    return !!p?.cloud && !this.acknowledged().includes(p.id);
  });
  readonly canSend = computed(
    () => !this.busy() && !!this.model() && !this.needsDisclosure() && this.pending().length === 0,
  );

  private approvedCalls: string[] = [];
  private deniedCalls: string[] = [];
  private abort: AbortController | null = null;

  async load(): Promise<void> {
    try {
      const meta = await this.api.meta();
      this.meta.set(meta);
      const first = meta.providers.find((p) => p.available && p.models.length > 0) ?? meta.providers[0];
      if (first) {
        this.selectProvider(first.id);
      }
    } catch (e) {
      this.error.set(errorText(e));
    }
  }

  selectProvider(id: string): void {
    this.providerId.set(id);
    this.model.set(this.provider()?.models[0] ?? '');
  }

  acknowledge(): void {
    const p = this.provider();
    if (p) {
      this.acknowledged.update((ids) => [...ids, p.id]);
    }
  }

  async send(text: string): Promise<void> {
    const content = text.trim();
    if (!content || !this.canSend()) {
      return;
    }
    this.messages.update((m) => [...m, { role: 'user', content }]);
    await this.runTurn();
  }

  /** Records a decision; once every pending call is decided, the turn continues. */
  async decide(callId: string, decision: Decision): Promise<void> {
    const call = this.pending().find((p) => p.call_id === callId);
    if (!call) {
      return;
    }
    if (decision === 'session' && call.writes) {
      decision = 'once'; // writes are never allowed for the whole session
    }
    this.decisions.update((d) => ({ ...d, [callId]: decision }));
    if (decision === 'deny') {
      this.deniedCalls.push(callId);
    } else {
      this.approvedCalls.push(callId);
      if (decision === 'session') {
        this.allowedTools.update((t) => (t.includes(call.tool) ? t : [...t, call.tool]));
      }
    }
    if (this.pending().every((p) => this.decisions()[p.call_id])) {
      this.pending.set([]);
      await this.runTurn();
    }
  }

  /** Aborts the running turn; finished messages stay, see runTurn. */
  cancel(): void {
    this.abort?.abort();
  }

  private async runTurn(): Promise<void> {
    const abort = new AbortController();
    this.abort = abort;
    this.busy.set(true);
    this.error.set('');
    this.streaming.set('');
    this.thinking.set('');
    let finished = false;
    try {
      await this.api.turnStream(
        {
          provider: this.providerId(),
          model: this.model(),
          messages: this.messages(),
          allowed_tools: this.allowedTools(),
          approved_calls: this.approvedCalls,
          denied_calls: this.deniedCalls,
          disclosure_ack: this.acknowledged().includes(this.providerId()),
        },
        (event) => {
          switch (event.type) {
            case 'delta':
              this.streaming.update((t) => t + event.text);
              break;
            case 'thinking':
              this.thinking.update((t) => t + event.text);
              break;
            case 'message':
              this.messages.update((m) => [...m, event.message]);
              this.streaming.set('');
              this.thinking.set('');
              break;
            case 'done':
              this.messages.set(event.messages);
              this.pending.set(event.pending ?? []);
              finished = true;
              break;
            case 'error':
              throw new Error(event.error);
          }
        },
        abort.signal,
      );
      if (!finished) {
        throw new Error('Die Verbindung endete, bevor der Zug abgeschlossen war.');
      }
    } catch (e) {
      this.keepFinished();
      if (abort.signal.aborted) {
        this.error.set('Abgebrochen.');
      } else {
        if (e instanceof ApiError && e.status === 428) {
          this.acknowledged.update((ids) => ids.filter((id) => id !== this.providerId()));
        }
        this.error.set(errorText(e));
      }
    } finally {
      this.streaming.set('');
      this.thinking.set('');
      this.busy.set(false);
      if (this.abort === abort) {
        this.abort = null;
      }
    }
  }

  /** After an abort or error: drop a tool call without result, keep text the model had written so far. */
  private keepFinished(): void {
    const partial = this.streaming().trim();
    this.messages.update((m) => {
      const kept = trimOpenCalls(m);
      return partial ? [...kept, { role: 'assistant', content: this.streaming() }] : kept;
    });
  }

  reset(): void {
    this.messages.set([]);
    this.pending.set([]);
    this.decisions.set({});
    this.approvedCalls = [];
    this.deniedCalls = [];
    this.error.set('');
  }
}

/**
 * Removes the last assistant message with tool calls that have no result, and everything after it, so no provider
 * sees an incomplete conversation.
 */
export function trimOpenCalls(messages: Message[]): Message[] {
  for (let i = messages.length - 1; i >= 0; i--) {
    const calls = messages[i].tool_calls ?? [];
    if (messages[i].role !== 'assistant' || calls.length === 0) {
      continue;
    }
    const answered = new Set(messages.slice(i + 1).filter((m) => m.role === 'tool').map((m) => m.tool_call_id));
    return calls.every((c) => answered.has(c.id)) ? messages : messages.slice(0, i);
  }
  return messages;
}

function errorText(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
