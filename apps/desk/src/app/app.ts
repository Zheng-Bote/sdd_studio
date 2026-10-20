import { Component, inject, viewChild } from '@angular/core';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { ChatPanel } from './chat/chat-panel';
import { ChatSession } from './core/chat-session';
import { ProcessPanel } from './process/process-panel';

/** Shell: provider and model selection, process sidebar, conversation. */
@Component({
  selector: 'desk-root',
  imports: [ChatPanel, ProcessPanel, HlmButtonImports, HlmBadgeImports],
  host: { class: 'flex h-dvh flex-col bg-background text-foreground' },
  template: `
    <header class="flex flex-wrap items-center gap-3 border-b px-4 py-2">
      <h1 class="text-base font-semibold">SDD Studio <span class="font-normal text-muted-foreground">Desktop</span></h1>
      <span class="text-sm text-muted-foreground">{{ session.meta()?.project }}</span>
      <div class="ml-auto flex flex-wrap items-center gap-2">
        <label class="flex items-center gap-1.5 text-sm">
          Anbieter
          <select class="h-8 rounded-md border bg-background px-2 text-sm" [value]="session.providerId()" (change)="session.selectProvider($any($event.target).value)">
            @for (p of session.meta()?.providers ?? []; track p.id) {
              <option [value]="p.id" [disabled]="!p.available">{{ p.name }}{{ p.cloud ? ' (Cloud)' : '' }}</option>
            }
          </select>
        </label>
        <label class="flex items-center gap-1.5 text-sm">
          Modell
          <select class="h-8 max-w-56 rounded-md border bg-background px-2 text-sm" [value]="session.model()" (change)="session.model.set($any($event.target).value)">
            @for (m of session.provider()?.models ?? []; track m) {
              <option [value]="m">{{ m }}</option>
            }
          </select>
        </label>
        @if (session.provider()?.cloud) {
          <span hlmBadge variant="outline" title="Inhalte gehen an diesen Anbieter">Cloud</span>
        }
        <button hlmBtn variant="outline" size="sm" type="button" (click)="session.reset()">Neues Gespräch</button>
      </div>
    </header>
    @for (w of session.meta()?.warnings ?? []; track w) {
      <p class="border-b bg-muted px-4 py-1 text-xs text-muted-foreground">{{ w }}</p>
    }
    <div class="flex min-h-0 flex-1">
      <aside class="w-80 shrink-0 overflow-y-auto border-r max-md:hidden" aria-label="Prozess">
        <desk-process-panel (started)="chat()?.prefill($event)" />
      </aside>
      <main class="flex min-h-0 flex-1 flex-col"><desk-chat-panel /></main>
    </div>
  `,
})
export class App {
  protected readonly session = inject(ChatSession);
  protected readonly chat = viewChild(ChatPanel);

  constructor() {
    const dark = globalThis.matchMedia?.('(prefers-color-scheme: dark)');
    const apply = (): void => {
      document.documentElement.classList.toggle('dark', !!dark?.matches);
    };
    apply();
    dark?.addEventListener('change', apply);
    void this.session.load();
  }
}
