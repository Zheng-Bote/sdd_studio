import { TestBed } from '@angular/core/testing';
import { DeskApi } from '../core/desk-api';
import { Guide, Step } from '../core/types';
import { ProcessPanel } from './process-panel';

const step = (id: string, title: string): Step => ({ id, title, state: 'open', detail: 'Details ' + id, command: 'befehl ' + id });

async function renderElement(guide: Guide): Promise<HTMLElement> {
  TestBed.resetTestingModule();
  TestBed.configureTestingModule({
    providers: [{ provide: DeskApi, useValue: { next: async () => guide, prompts: async () => ({ prompts: [] }) } }],
  });
  const fixture = TestBed.createComponent(ProcessPanel);
  await fixture.whenStable();
  fixture.detectChanges();
  return fixture.nativeElement as HTMLElement;
}

function texts(el: HTMLElement, testid: string): string[] {
  return Array.from(el.querySelectorAll(`[data-testid="${testid}"]`)).map((e) => e.textContent ?? '');
}

describe('ProcessPanel', () => {
  // desk-project-steps AC-1: project steps appear in the order of sdd next, with state, title, detail and command.
  it('shows the project steps in order', async () => {
    const el = await renderElement({
      features: [],
      architecture: step('architecture', 'Architekturdoku nachführen'),
      adoption: step('adoption', 'Übernahme abschließen'),
      constitution: step('constitution', 'Verfassung ausfüllen'),
    });
    const shown = texts(el, 'project-step');
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
    expect(texts(await renderElement({ features: [] }), 'project-step')).toEqual([]);
    const el = await renderElement({ features: [], architecture: step('architecture', 'Architekturdoku anlegen') });
    expect(texts(el, 'project-step').length).toBe(1);
  });

  // bmad-import AC-7: the sidebar separates "Vorhaben" from "Features"; the initiative appears on the macro floor.
  it('separates the macro floor from the features', async () => {
    const el = await renderElement({
      features: [{ feature: 'epic-plattform', steps: [], next: step('epic-refine', 'Epic verfeinern') }],
      initiative: { id: 'initiative', title: 'Initiative', state: 'done', detail: 'docs/initiative.md, 31 Anforderungen' },
    });
    const headings = Array.from(el.querySelectorAll('h2')).map((h) => (h.textContent ?? '').trim());
    expect(headings).toEqual(['Geführte Abläufe', 'Vorhaben', 'Features']);
    expect(texts(el, 'initiative-step')[0]).toContain('31 Anforderungen');
    const features = Array.from(el.querySelectorAll('[data-testid="feature"]')).map((f) => f.textContent ?? '');
    expect(features.length).toBe(1);
    expect(features[0]).toContain('epic-plattform');
    expect(features[0]).toContain('Epic verfeinern');
  });

  // bmad-import AC-7: without a macro level the "Vorhaben" floor stays hidden.
  it('hides the macro floor without a macro step', async () => {
    const el = await renderElement({ features: [] });
    const headings = Array.from(el.querySelectorAll('h2')).map((h) => (h.textContent ?? '').trim());
    expect(headings).toEqual(['Geführte Abläufe', 'Features']);
    expect(texts(el, 'initiative-step')).toEqual([]);
  });
});
