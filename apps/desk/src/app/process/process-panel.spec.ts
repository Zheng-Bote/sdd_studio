import { TestBed } from '@angular/core/testing';
import { DeskApi } from '../core/desk-api';
import { Guide, Step } from '../core/types';
import { ProcessPanel } from './process-panel';

const step = (id: string, title: string): Step => ({ id, title, state: 'open', detail: 'Details ' + id, command: 'befehl ' + id });

async function render(guide: Guide) {
  TestBed.configureTestingModule({
    providers: [{ provide: DeskApi, useValue: { next: async () => guide, prompts: async () => ({ prompts: [] }) } }],
  });
  const fixture = TestBed.createComponent(ProcessPanel);
  await fixture.whenStable();
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  return Array.from(el.querySelectorAll('[data-testid="project-step"]')).map((e) => e.textContent ?? '');
}

describe('ProcessPanel', () => {
  // desk-project-steps AC-1: project steps appear in the order of sdd next, with state, title, detail and command.
  it('shows the project steps in order', async () => {
    const shown = await render({
      features: [],
      architecture: step('architecture', 'Architekturdoku nachführen'),
      adoption: step('adoption', 'Übernahme abschließen'),
      constitution: step('constitution', 'Verfassung ausfüllen'),
    });
    expect(shown.length).toBe(3);
    expect(shown[0]).toContain('Verfassung ausfüllen');
    expect(shown[1]).toContain('Übernahme abschließen');
    expect(shown[2]).toContain('Architekturdoku nachführen');
    expect(shown[2]).toContain('offen');
    expect(shown[2]).toContain('Details architecture');
    expect(shown[2]).toContain('befehl architecture');
  });

  // desk-project-steps AC-2: a step sdd next does not report is not shown.
  it('shows nothing for missing project steps', async () => {
    expect(await render({ features: [] })).toEqual([]);
    TestBed.resetTestingModule();
    const shown = await render({ features: [], architecture: step('architecture', 'Architekturdoku anlegen') });
    expect(shown.length).toBe(1);
  });
});
