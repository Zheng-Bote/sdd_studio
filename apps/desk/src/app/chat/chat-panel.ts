import { Component, ElementRef, effect, inject, signal, viewChild } from '@angular/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { HlmTextareaImports } from '@spartan-ng/helm/textarea';
import { ChatSession } from '../core/chat-session';
import { ApprovalCard } from './approval-card';
import { MessageView } from './message-view';

/** Conversation, pending approvals, cloud disclosure and the composer. */
@Component({
  selector: 'desk-chat-panel',
  imports: [MessageView, ApprovalCard, HlmButtonImports, HlmTextareaImports, HlmAlertImports, HlmSpinnerImports],
  host: { class: 'flex min-h-0 flex-1 flex-col' },
  template: `
    <div #log class="flex-1 space-y-3 overflow-y-auto p-4" role="log" aria-live="polite" aria-label="Gespräch">
      @if (session.messages().length === 0) {
        <div class="mx-auto mt-16 max-w-md text-center text-muted-foreground">
          <p class="text-lg font-medium text-foreground">Wobei soll ich helfen?</p>
          <p class="mt-2 text-sm">Beschreiben Sie eine Idee oder starten Sie links einen geführten Ablauf — von der Idee zum PRD bis zur Prüfung.</p>
        </div>
      }
      @for (m of session.messages(); track $index) {
        <desk-message [message]="m" />
      }
      @for (p of session.pending(); track p.call_id) {
        <desk-approval-card [pending]="p" (decided)="session.decide(p.call_id, $event)" />
      }
      @if (session.busy()) {
        <div class="flex items-center gap-2 text-sm text-muted-foreground"><hlm-spinner /> denkt nach …</div>
      }
    </div>

    @if (session.needsDisclosure()) {
      <div hlmAlert class="mx-4 mb-3 border-amber-500/60">
        <h4 hlmAlertTitle>Inhalte verlassen dieses Gerät</h4>
        <p hlmAlertDescription>
          Sie haben {{ session.provider()?.name }} gewählt. Ihre Nachrichten und die Ergebnisse der sdd-Werkzeuge
          (z. B. Inhalte Ihrer Spezifikationen) gehen ungeschwärzt an diesen Anbieter.
        </p>
        <div class="mt-2">
          <button hlmBtn size="sm" type="button" (click)="session.acknowledge()">Verstanden, {{ session.provider()?.name }} verwenden</button>
        </div>
      </div>
    }
    @if (session.error(); as err) {
      <div hlmAlert variant="destructive" class="mx-4 mb-3" role="alert">
        <p hlmAlertDescription>{{ err }}</p>
      </div>
    }

    <form class="flex items-end gap-2 border-t p-3" (submit)="submit($event)">
      <label for="composer" class="sr-only">Nachricht</label>
      <textarea
        hlmTextarea
        id="composer"
        rows="3"
        class="min-h-16 flex-1 resize-y"
        placeholder="Nachricht … (Strg+Enter sendet)"
        [value]="draft()"
        (input)="draft.set($any($event.target).value)"
        (keydown.control.enter)="submit($event)"
        (keydown.meta.enter)="submit($event)"
      ></textarea>
      <button hlmBtn type="submit" [disabled]="!session.canSend() || !draft().trim()">Senden</button>
    </form>
  `,
})
export class ChatPanel {
  protected readonly session = inject(ChatSession);
  protected readonly draft = signal('');
  private readonly log = viewChild<ElementRef<HTMLElement>>('log');

  constructor() {
    effect(() => {
      this.session.messages();
      this.session.pending();
      const el = this.log()?.nativeElement;
      queueMicrotask(() => el?.scrollTo({ top: el.scrollHeight }));
    });
  }

  protected async submit(event: Event): Promise<void> {
    event.preventDefault();
    const text = this.draft();
    if (!text.trim() || !this.session.canSend()) {
      return;
    }
    this.draft.set('');
    await this.session.send(text);
  }

  /** Puts a guided prompt into the composer so the user can read and adjust it before sending. */
  prefill(text: string): void {
    this.draft.set(text);
  }
}
