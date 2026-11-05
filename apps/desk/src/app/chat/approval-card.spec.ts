import { TestBed } from '@angular/core/testing';
import { ApprovalCard } from './approval-card';
import { Decision, Pending } from '../core/types';

function render(pending: Pending) {
  const fixture = TestBed.createComponent(ApprovalCard);
  fixture.componentRef.setInput('pending', pending);
  const decisions: Decision[] = [];
  fixture.componentInstance.decided.subscribe((d) => decisions.push(d));
  fixture.detectChanges();
  const el = fixture.nativeElement as HTMLElement;
  const buttons = Array.from(el.querySelectorAll('button')).map((b) => b.textContent?.trim());
  return { el, buttons, decisions };
}

describe('ApprovalCard', () => {
  // sdd-desk AC-4: a write shows its preview before it runs.
  // desk-write-content AC-7: writes may be allowed for the session, too.
  it('shows the preview of a write and offers the session approval', () => {
    const { el, buttons } = render({
      call_id: 'w1',
      tool: 'sdd_new',
      arguments: { feature: 'billing', apply: true },
      writes: true,
      preview: { ops: [{ path: 'docs/prd/billing.md', action: 'create', content: 'feature: billing' }] },
    });
    expect(el.textContent).toContain('schreibt Dateien');
    expect(el.textContent).toContain('docs/prd/billing.md');
    expect(el.textContent).toContain('feature: billing');
    expect(el.textContent).toContain('ohne weitere Rückfrage');
    expect(buttons).toEqual(['Einmal erlauben', 'Für diese Sitzung erlauben', 'Ablehnen']);
  });

  // desk-approval-scope AC-4: a preview of a write tool is marked and says that a session approval covers previews only.
  it('marks a preview of a write tool as preview only', () => {
    const { el, buttons } = render({ call_id: 'v1', tool: 'sdd_new', arguments: { feature: 'billing' }, writes: false, preview_only: true });
    expect(el.textContent).toContain('nur Vorschau');
    expect(el.textContent).toContain('Schreiben wird erneut gefragt');
    expect(el.textContent).not.toContain('nur lesend');
    expect(buttons).toEqual(['Einmal erlauben', 'Für diese Sitzung erlauben', 'Ablehnen']);
  });

  // desk-write-content AC-7: an update of sdd_write shows its diff, labelled as a change.
  it('shows the diff of an update', () => {
    const { el } = render({
      call_id: 'w2',
      tool: 'sdd_write',
      arguments: { path: 'docs/prd/billing.md', apply: true },
      writes: true,
      preview: { ops: [{ path: 'docs/prd/billing.md', action: 'update', diff: '-- AC-2 Import\n+- AC-2 Import aus CSV\n' }] },
    });
    expect(el.textContent).toContain('ändern');
    expect(el.querySelector('pre[aria-label="Änderungen"]')?.textContent).toContain('+- AC-2 Import aus CSV');
  });

  // sdd-archive AC-4: a move shows source, target and the frontmatter diff.
  it('shows source, target and diff of a move', () => {
    const { el } = render({
      call_id: 'a1',
      tool: 'sdd_archive',
      arguments: { feature: 'billing', apply: true },
      writes: true,
      preview: { ops: [{ path: 'docs/archive/prd/billing.md', action: 'move', from: 'docs/prd/billing.md', diff: '-status: active\n+status: archived\n' }] },
    });
    expect(el.textContent).toContain('verschieben');
    expect(el.textContent).toContain('docs/prd/billing.md');
    expect(el.textContent).toContain('docs/archive/prd/billing.md');
    expect(el.querySelector('pre[aria-label="Änderungen"]')?.textContent).toContain('+status: archived');
  });

  // desk-critical-tools AC-5: a critical action is marked, names its effect on the button and has no session approval.
  // desk-config-critical AC-6: a configuration change shows the findings it drops and adds, confirmed singly.
  it('shows the effect of a configuration change on the findings', () => {
    const { el, buttons } = render({
      call_id: 'k2',
      tool: 'sdd_config',
      arguments: { apply: true },
      writes: true,
      critical: true,
      summary: 'sdd.config.yaml ändern: exclude (−1 / +0 Befunde)',
      preview: {
        ops: [{ path: 'sdd.config.yaml', action: 'update', diff: '+  - docs/prd/neu.md\n' }],
        findings_delta: [{ gate: 'lint', dropped: [{ rule_id: 'SDD-022', path: 'docs/prd/neu.md', line: 12 }], added: [] }],
        unchecked: [{ gate: 'pre-merge', error: 'kein Git-Repository' }],
      },
    });
    const effect = el.querySelector('[aria-label="Wirkung auf die Befunde"]')?.textContent ?? '';
    expect(effect).toContain('fällt weg');
    expect(effect).toContain('SDD-022 docs/prd/neu.md:12');
    expect(effect).toContain('pre-merge: nicht verglichen');
    expect(el.textContent).toContain('kritisch');
    expect(buttons).toEqual(['sdd.config.yaml ändern: exclude (−1 / +0 Befunde)', 'Ablehnen']);
  });

  it('asks for a single confirmation of a critical action', () => {
    const { el, buttons, decisions } = render({
      call_id: 'k1',
      tool: 'sdd_baseline',
      arguments: { apply: true },
      writes: true,
      critical: true,
      summary: 'Baseline: 1 Einträge aufnehmen (SDD-022 ×1), 0 entfernen',
      preview: { added: [{ gate: 'pre-merge', rule_id: 'SDD-022', path: 'docs/prd/a.md', line: 12 }], locked: ['docs/adr/0001-x.md'] },
    });
    expect(el.textContent).toContain('kritisch');
    expect(el.textContent).toContain('nur dieses eine Mal');
    expect(el.textContent).toContain('SDD-022 docs/prd/a.md:12');
    expect(el.textContent).toContain('docs/adr/0001-x.md');
    expect(buttons).toEqual(['Baseline: 1 Einträge aufnehmen (SDD-022 ×1), 0 entfernen', 'Ablehnen']);
    (el.querySelectorAll('button')[0] as HTMLButtonElement).click();
    expect(decisions).toEqual(['once']);
  });

  it('offers once, session and deny for a read-only call and emits the choice', () => {
    const { el, buttons, decisions } = render({ call_id: 'c1', tool: 'sdd_next', arguments: {}, writes: false });
    expect(buttons).toEqual(['Einmal erlauben', 'Für diese Sitzung erlauben', 'Ablehnen']);
    (el.querySelectorAll('button')[1] as HTMLButtonElement).click();
    (el.querySelectorAll('button')[2] as HTMLButtonElement).click();
    expect(decisions).toEqual(['session', 'deny']);
  });

  it('renders tool arguments as text, never as HTML', () => {
    const { el } = render({ call_id: 'x', tool: 'sdd_explain', arguments: { rule_id: '<img src=x onerror=alert(1)>' }, writes: false });
    expect(el.querySelector('img')).toBeNull();
    expect(el.textContent).toContain('<img src=x onerror=alert(1)>');
  });
});
