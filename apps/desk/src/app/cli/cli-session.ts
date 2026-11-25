import { computed, inject, Service, signal } from '@angular/core';
import { ApiError, DeskApi } from '../core/desk-api';
import { CliCard, CliDecision, CliInfo } from '../core/types';

/**
 * One line describing a CLI in the selection: offered, or the reason it is not (agenten-cli-proxy AC-1). The reason
 * comes from the backend in plain words, so a CLI that is missing is missing for a stated reason.
 */
export function cliLabel(c: CliInfo): string {
  return c.offerable ? c.name : `${c.name} — nicht angeboten: ${c.blocked_because ?? 'Grund unbekannt'}`;
}

/** The recorded state in one phrase; a CLI without a record is "nicht belegt", never "belegt" (AC-8). */
export function cliStatusText(c: CliInfo): string {
  if (c.evidence.status === 'failed') {
    return 'im Test durchgefallen';
  }
  if (c.evidence.status !== 'proven' || !c.evidence.proxy_proven) {
    return 'nicht belegt';
  }
  return 'belegt';
}

/** The tier of a card in words. The tier decides what a session allowance may cover. */
export function tierText(tier: CliCard['tier']): string {
  return tier === 'critical' ? 'kritisch' : tier === 'write' ? 'schreibend' : 'lesend';
}

/** A session allowance is offered only where the backend accepts it: never for a critical call (AC-2). */
export function sessionDecision(card: CliCard): CliDecision | null {
  if (card.tier === 'critical') {
    return null;
  }
  return card.tier === 'write' ? 'allow_write_session' : 'allow_tool_session';
}

/** How many seconds a card has left before it expires; 0 when the deadline has passed (AC-4). */
export function secondsLeft(card: CliCard, now: number): number {
  const left = new Date(card.deadline).getTime() - now;
  return left > 0 ? Math.ceil(left / 1000) : 0;
}

/**
 * State of the agent CLI mode. The backend owns the sessions and the register; this holds what the user sees and
 * polls the open cards while a session runs (AC-3, AC-10).
 */
@Service()
export class CliSession {
  private readonly api = inject(DeskApi);

  readonly clis = signal<CliInfo[]>([]);
  readonly cloudNotice = signal('');
  readonly cards = signal<CliCard[]>([]);
  readonly selected = signal('');
  readonly prompt = signal('');
  readonly acknowledged = signal(false);
  readonly session = signal('');
  readonly busy = signal(false);
  readonly error = signal('');

  /** Only an offered CLI is selectable; everything else says why it is not. */
  readonly offerable = computed(() => this.clis().filter((c) => c.offerable));

  private timer: ReturnType<typeof setInterval> | null = null;

  async load(): Promise<void> {
    try {
      const list = await this.api.clis();
      this.clis.set(list.clis);
      this.cloudNotice.set(list.cloud_notice);
      const first = this.offerable()[0];
      if (!this.selected() && first) {
        this.selected.set(first.id);
      }
    } catch (err) {
      this.error.set(err instanceof ApiError ? err.message : String(err));
    }
    await this.refreshCards();
  }

  async refreshCards(): Promise<void> {
    try {
      this.cards.set((await this.api.cliCards()).cards);
    } catch (err) {
      this.error.set(err instanceof ApiError ? err.message : String(err));
    }
    this.syncPolling();
  }

  /** Starts polling only while someone may decide; an idle panel asks the backend nothing. */
  private syncPolling(): void {
    const active = this.cards().length > 0 || this.session() !== '';
    if (active && this.timer === null) {
      this.timer = setInterval(() => void this.refreshCards(), 2000);
    }
    if (!active && this.timer !== null) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }

  async start(): Promise<void> {
    const cli = this.selected();
    if (!cli || !this.acknowledged() || !this.prompt().trim()) {
      return;
    }
    this.busy.set(true);
    this.error.set('');
    try {
      const started = await this.api.startCli(cli, this.prompt().trim());
      this.session.set(started.session);
      await this.refreshCards();
    } catch (err) {
      this.error.set(err instanceof ApiError ? err.message : String(err));
    } finally {
      this.busy.set(false);
    }
  }

  async decide(card: CliCard, decision: CliDecision): Promise<void> {
    this.error.set('');
    try {
      await this.api.decideCli(card.call_id, decision);
    } catch (err) {
      this.error.set(err instanceof ApiError ? err.message : String(err));
    }
    await this.refreshCards();
  }

  stop(): void {
    if (this.timer !== null) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }
}
