import { Component, computed, input, output } from '@angular/core';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { Decision, Pending } from '../core/types';

/** Asks for permission to run one tool call; writes show their full preview and cannot be allowed per session. */
@Component({
  selector: 'desk-approval-card',
  imports: [HlmCardImports, HlmButtonImports, HlmBadgeImports],
  template: `
    <section hlmCard class="border-amber-500/60" [attr.aria-labelledby]="'approval-' + pending().call_id">
      <div hlmCardHeader>
        <h3 hlmCardTitle [id]="'approval-' + pending().call_id">
          Werkzeug <code>{{ pending().tool }}</code> ausführen?
        </h3>
        <p hlmCardDescription>
          @if (pending().writes) {
            <span hlmBadge variant="destructive">schreibt Dateien</span>
            Erst nach Ihrer Freigabe; geprüft wird nur diese eine Ausführung.
          } @else {
            <span hlmBadge variant="secondary">nur lesend</span>
          }
        </p>
      </div>
      <div hlmCardContent class="space-y-3">
        <details>
          <summary class="cursor-pointer text-sm text-muted-foreground">Argumente</summary>
          <pre class="mt-2 max-h-48 overflow-auto rounded-md bg-muted p-2 text-xs">{{ args() }}</pre>
        </details>
        @if (pending().preview?.error; as err) {
          <p class="text-sm text-destructive">Vorschau fehlgeschlagen: {{ err }}</p>
        }
        @for (op of pending().preview?.ops ?? []; track op.path) {
          <div class="rounded-md border">
            <div class="flex items-center gap-2 border-b px-3 py-1.5 text-sm">
              <span hlmBadge [variant]="op.action === 'skip' ? 'outline' : 'default'">
                {{ op.action === 'skip' ? 'übersprungen' : 'anlegen' }}
              </span>
              <code>{{ op.path }}</code>
              @if (op.reason) {
                <span class="text-muted-foreground">— {{ op.reason }}</span>
              }
            </div>
            @if (op.content) {
              <pre class="max-h-64 overflow-auto p-3 text-xs">{{ op.content }}</pre>
            }
          </div>
        }
      </div>
      <div hlmCardFooter class="flex flex-wrap gap-2">
        <button hlmBtn type="button" (click)="decided.emit('once')">Einmal erlauben</button>
        @if (!pending().writes) {
          <button hlmBtn variant="secondary" type="button" (click)="decided.emit('session')">
            Für diese Sitzung erlauben
          </button>
        }
        <button hlmBtn variant="destructive" type="button" (click)="decided.emit('deny')">Ablehnen</button>
      </div>
    </section>
  `,
})
export class ApprovalCard {
  readonly pending = input.required<Pending>();
  readonly decided = output<Decision>();
  protected readonly args = computed(() => JSON.stringify(this.pending().arguments, null, 2));
}
