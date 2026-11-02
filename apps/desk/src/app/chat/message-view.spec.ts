import { TestBed } from '@angular/core/testing';
import { MessageView } from './message-view';
import { Message } from '../core/types';

function render(message: Message): HTMLElement {
  const fixture = TestBed.createComponent(MessageView);
  fixture.componentRef.setInput('message', message);
  fixture.detectChanges();
  return fixture.nativeElement as HTMLElement;
}

describe('MessageView', () => {
  // flow-closing AC-2: an answer without text and without tool calls is shown as "stopped without an answer".
  it('marks an answer without text and tool calls', () => {
    const el = render({ role: 'assistant', content: '  ' });
    expect(el.querySelector('[role="note"]')?.textContent).toContain('ohne abschließende Antwort aufgehört');
  });

  it('shows no note for a text answer or a pure tool call', () => {
    expect(render({ role: 'assistant', content: 'Fertig.' }).querySelector('[role="note"]')).toBeNull();
    const call = render({ role: 'assistant', content: '', tool_calls: [{ id: 'c1', name: 'sdd_next', arguments: {} }] });
    expect(call.querySelector('[role="note"]')).toBeNull();
    expect(call.textContent).toContain('sdd_next');
  });
});
