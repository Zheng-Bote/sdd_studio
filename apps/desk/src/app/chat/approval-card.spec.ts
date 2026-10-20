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
  // sdd-desk AC-4: a write shows its preview and offers no session-wide approval.
  it('shows the preview of a write and no session approval', () => {
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
    expect(buttons).toEqual(['Einmal erlauben', 'Ablehnen']);
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
