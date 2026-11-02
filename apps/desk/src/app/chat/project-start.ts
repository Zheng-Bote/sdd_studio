import { Component, computed, input, output } from '@angular/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { ProjectState } from '../core/types';

/** Suggests how to start when the opened project is not covered by sdd yet; nothing for a configured project. */
@Component({
  selector: 'desk-project-start',
  imports: [HlmAlertImports, HlmButtonImports],
  template: `
    @if (offer(); as o) {
      <div hlmAlert class="border-sky-500/60" role="status">
        <h4 hlmAlertTitle>Dieses Projekt ist noch nicht mit sdd erfasst.</h4>
        <p hlmAlertDescription>{{ o.text }}</p>
        <div class="mt-2">
          <button hlmBtn size="sm" type="button" (click)="start.emit(o.prompt)">{{ o.label }}</button>
        </div>
      </div>
    }
  `,
})
export class ProjectStart {
  readonly state = input<ProjectState | undefined>();
  /** The name of the guided prompt to put into the composer. */
  readonly start = output<string>();

  protected readonly offer = computed(() => {
    switch (this.state()) {
      case 'existing':
        return {
          prompt: 'sdd-adopt',
          label: 'Bestehendes Projekt übernehmen',
          text: 'Der Ordner enthält schon Dateien. Die Übernahme zeigt zuerst einen Bericht und legt dann auf einem eigenen Branch Konfiguration, Verfassung und Agenten-Dateien an; danach erarbeiten wir die Verfassung.',
        };
      case 'empty':
        return {
          prompt: 'sdd-start',
          label: 'Neues Projekt anlegen',
          text: 'Der Ordner ist leer. Wir legen das Projekt mit Git an, klären die Technik, erarbeiten die Verfassung und beginnen das erste Feature.',
        };
      default:
        return null;
    }
  });
}
