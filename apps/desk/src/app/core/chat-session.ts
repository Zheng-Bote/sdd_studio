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

  private async runTurn(): Promise<void> {
    this.busy.set(true);
    this.error.set('');
    try {
      const res = await this.api.turn({
        provider: this.providerId(),
        model: this.model(),
        messages: this.messages(),
        allowed_tools: this.allowedTools(),
        approved_calls: this.approvedCalls,
        denied_calls: this.deniedCalls,
        disclosure_ack: this.acknowledged().includes(this.providerId()),
      });
      this.messages.set(res.messages);
      this.pending.set(res.pending);
    } catch (e) {
      if (e instanceof ApiError && e.status === 428) {
        this.acknowledged.update((ids) => ids.filter((id) => id !== this.providerId()));
      }
      this.error.set(errorText(e));
    } finally {
      this.busy.set(false);
    }
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

function errorText(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
