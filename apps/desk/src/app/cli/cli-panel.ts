import { Component, OnDestroy, inject } from '@angular/core';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { CliCard } from '../core/types';
import { CliSession, cliLabel, cliStatusText, secondsLeft, sessionDecision, tierText } from './cli-session';

/**
 * The agent CLI mode: which CLI is offered and why one is not, the cloud notice before the first run, and the cards
 * of the running sessions (agenten-cli-proxy AC-1, AC-6, AC-10). Every card names its CLI and its session, so two
 * running CLIs stay distinguishable.
 */
@Component({
  selector: 'desk-cli-panel',
  imports: [HlmButtonImports, HlmBadgeImports],
  template: `
    <section class="border-t p-3 text-sm" aria-label="Agenten-CLI">
      <div class="mb-2 flex items-center gap-2">
        <h2 class="font-medium">Agenten-CLI</h2>
        @if (cli.session()) {
          <span hlmBadge variant="outline" title="Sitzungskennung">Sitzung {{ cli.session().slice(0, 8) }}</span>
        }
      </div>

      @if (cli.offerable().length === 0) {
        <p class="text-xs text-muted-foreground">
          Keine Agenten-CLI angeboten. Eine CLI erscheint erst, wenn Startvertrag, Schreibsperre und der Lauf gegen den
          Proxy belegt sind.
        </p>
      } @else {
        <label class="mb-2 block">
          <span class="text-xs text-muted-foreground">CLI</span>
          <select
            class="h-8 w-full rounded-md border bg-background px-2 text-sm"
            [value]="cli.selected()"
            (change)="cli.selected.set($any($event.target).value)"
          >
            @for (c of cli.offerable(); track c.id) {
              <option [value]="c.id">{{ c.name }}{{ c.cloud ? ' (Cloud)' : '' }}</option>
            }
          </select>
        </label>

        @for (c of cli.clis(); track c.id) {
          @if (c.id === cli.selected()) {
            <p class="mb-2 text-xs" [class]="evidenceClass(c.evidence.status)">
              Belegstand: {{ statusText(c) }}
              @if (c.evidence.note) {
                — {{ c.evidence.note }}
              }
            </p>
          }
        }

        <label class="mb-2 flex items-start gap-2 text-xs">
          <input
            type="checkbox"
            class="mt-0.5"
            [checked]="cli.acknowledged()"
            (change)="cli.acknowledged.set($any($event.target).checked)"
          />
          <span>{{ cli.cloudNotice() || 'Inhalte verlassen das Gerät.' }}</span>
        </label>

        <textarea
          class="mb-2 min-h-16 w-full rounded-md border bg-background px-2 py-1 text-sm"
          placeholder="Auftrag an die CLI, z. B. „Prüfe den Prozessstand und schlage den nächsten Schritt vor.“"
          [value]="cli.prompt()"
          (input)="cli.prompt.set($any($event.target).value)"
        ></textarea>

        <button
          hlmBtn
          variant="outline"
          size="sm"
          type="button"
          class="w-full"
          [disabled]="!cli.acknowledged() || !cli.prompt().trim() || cli.busy()"
          [title]="cli.acknowledged() ? 'CLI-Sitzung starten' : 'Erst den Cloud-Hinweis bestätigen'"
          (click)="cli.start()"
        >
          CLI starten
        </button>

        @if (apiError(); as err) {
          <p class="mt-2 text-xs text-destructive" role="alert">{{ err }}</p>
        }

        @if (cli.cards().length > 0) {
          <h3 class="mb-1 mt-3 text-xs font-medium uppercase text-muted-foreground">
            Offene Aufrufe ({{ cli.cards().length }})
          </h3>
          @for (card of cli.cards(); track card.call_id) {
            <div class="mb-2 rounded-md border p-2">
              <div class="flex items-center gap-1.5">
                <span class="font-mono text-xs">{{ card.tool }}</span>
                <span hlmBadge variant="outline">{{ tierLabel(card.tier) }}</span>
                <span hlmBadge variant="outline" title="CLI und Sitzung dieser Karte">{{ card.cli }}</span>
              </div>
              <p class="text-xs text-muted-foreground">
                Kennung {{ card.call_id }} · Sitzung {{ card.session.slice(0, 8) }} ·
                @if (left(card) > 0) {
                  verfällt in {{ left(card) }} s
                } @else {
                  Frist abgelaufen
                }
              </p>
              @if (card.summary) {
                <p class="mt-1 text-xs">{{ card.summary }}</p>
              }
              <div class="mt-2 flex flex-wrap gap-1">
                <button hlmBtn size="sm" type="button" (click)="cli.decide(card, 'allow')">Einmal erlauben</button>
                @if (sessionDecision(card); as scope) {
                  <button hlmBtn variant="outline" size="sm" type="button" (click)="cli.decide(card, scope)">
                    Für die Sitzung
                  </button>
                }
                <button hlmBtn variant="ghost" size="sm" type="button" (click)="cli.decide(card, 'deny')">
                  Ablehnen
                </button>
              </div>
            </div>
          }
        }
      }

      <button hlmBtn variant="ghost" size="sm" type="button" class="mt-1 w-full" (click)="cli.load()">
        Liste und Karten aktualisieren
      </button>

      @for (c of cli.clis(); track c.id) {
        @if (!c.offerable) {
          <p class="mt-1 text-xs text-muted-foreground">{{ label(c) }}</p>
        }
      }
    </section>
  `,
})
export class CliPanel implements OnDestroy {
  protected readonly cli = inject(CliSession);
  protected readonly tierLabel = tierText;
  protected readonly sessionDecision = sessionDecision;
  protected readonly label = cliLabel;
  protected readonly statusText = cliStatusText;

  constructor() {
    void this.cli.load();
  }

  ngOnDestroy(): void {
    this.cli.stop();
  }

  /** Where the card's deadline stands; the card is useless once it passed, and says so (AC-4). */
  protected left(card: CliCard): number {
    return secondsLeft(card, Date.now());
  }

  protected evidenceClass(status: string): string {
    if (status === 'failed') {
      return 'text-destructive';
    }
    return status === 'proven' ? 'text-muted-foreground' : 'text-amber-700 dark:text-amber-400';
  }

  protected apiError(): string {
    return this.cli.error();
  }
}
