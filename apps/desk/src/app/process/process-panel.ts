import { Component, inject, output, signal } from '@angular/core';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmInputImports } from '@spartan-ng/helm/input';
import { HlmSeparatorImports } from '@spartan-ng/helm/separator';
import { DeskApi } from '../core/desk-api';
import { Guide, PromptInfo, Step } from '../core/types';

const STATE_LABEL: Record<Step['state'], string> = {
  done: 'erledigt',
  open: 'offen',
  manual: 'manuell',
  'not-needed': 'nicht nötig',
  later: 'später',
};

/** Process state from sdd next and the guided flows of sdd mcp. */
@Component({
  selector: 'desk-process-panel',
  imports: [HlmButtonImports, HlmBadgeImports, HlmInputImports, HlmSeparatorImports],
  host: { class: 'block space-y-4 p-4' },
  template: `
    <section aria-labelledby="flows-title" class="space-y-2">
      <h2 id="flows-title" class="text-sm font-semibold">Geführte Abläufe</h2>
      @for (p of prompts(); track p.name) {
        <div class="rounded-md border p-2">
          <div class="text-sm font-medium">{{ p.title }}</div>
          <p class="text-xs text-muted-foreground">{{ p.description }}</p>
          @for (a of p.arguments; track a.name) {
            <label class="mt-1 block text-xs">
              {{ a.description }}{{ a.required ? ' *' : '' }}
              <input hlmInput class="mt-0.5 h-7 w-full text-xs" [value]="value(p.name, a.name)" (input)="setValue(p.name, a.name, $any($event.target).value)" />
            </label>
          }
          <button hlmBtn size="sm" variant="secondary" class="mt-2" type="button" [disabled]="!ready(p)" (click)="start(p)">Starten</button>
        </div>
      }
    </section>
    <hlm-separator />
    <section aria-labelledby="state-title" class="space-y-2">
      <div class="flex items-center justify-between">
        <h2 id="state-title" class="text-sm font-semibold">Prozessstand</h2>
        <button hlmBtn size="xs" variant="ghost" type="button" (click)="refresh()">Aktualisieren</button>
      </div>
      @if (guide()?.start; as s) {
        <p class="text-sm">{{ s.title }}</p>
        <p class="text-xs text-muted-foreground">{{ s.detail }}</p>
      }
      @for (f of guide()?.features ?? []; track f.feature) {
        <div class="rounded-md border p-2">
          <div class="text-sm font-medium"><code>{{ f.feature }}</code></div>
          @if (f.next; as n) {
            <div class="mt-1 flex items-center gap-1.5 text-xs">
              <span hlmBadge [variant]="n.state === 'open' ? 'default' : 'secondary'">{{ label(n.state) }}</span>
              {{ n.title }}
            </div>
            <p class="mt-0.5 text-xs text-muted-foreground">{{ n.detail }}</p>
          }
        </div>
      }
      @if (error()) {
        <p class="text-xs text-destructive">{{ error() }}</p>
      }
    </section>
  `,
})
export class ProcessPanel {
  private readonly api = inject(DeskApi);
  readonly started = output<string>();
  protected readonly guide = signal<Guide | null>(null);
  protected readonly prompts = signal<PromptInfo[]>([]);
  protected readonly error = signal('');
  private readonly values = signal<Record<string, string>>({});

  constructor() {
    void this.refresh();
    this.api.prompts().then(
      (r) => this.prompts.set(r.prompts),
      (e: unknown) => this.error.set(String(e)),
    );
  }

  async refresh(): Promise<void> {
    try {
      this.guide.set(await this.api.next());
      this.error.set('');
    } catch (e) {
      this.error.set(e instanceof Error ? e.message : String(e));
    }
  }

  protected label(state: Step['state']): string {
    return STATE_LABEL[state];
  }

  protected value(prompt: string, arg: string): string {
    return this.values()[prompt + '/' + arg] ?? '';
  }

  protected setValue(prompt: string, arg: string, v: string): void {
    this.values.update((all) => ({ ...all, [prompt + '/' + arg]: v }));
  }

  protected ready(p: PromptInfo): boolean {
    return p.arguments.every((a) => !a.required || this.value(p.name, a.name).trim() !== '');
  }

  protected async start(p: PromptInfo): Promise<void> {
    const args: Record<string, string> = {};
    for (const a of p.arguments) {
      const v = this.value(p.name, a.name).trim();
      if (v) {
        args[a.name] = v;
      }
    }
    try {
      const res = await this.api.prompt(p.name, args);
      this.started.emit(res.messages.map((m) => m.content.text).join('\n\n'));
    } catch (e) {
      this.error.set(e instanceof Error ? e.message : String(e));
    }
  }
}
