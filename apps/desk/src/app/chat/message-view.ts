import { Component, computed, input } from '@angular/core';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { Message } from '../core/types';

/** One conversation entry; text is rendered as text, never as HTML. */
@Component({
  selector: 'desk-message',
  imports: [HlmBadgeImports],
  host: { class: 'block' },
  template: `
    @switch (message().role) {
      @case ('user') {
        <div class="ml-auto max-w-[80%] rounded-lg bg-primary px-4 py-2 text-primary-foreground whitespace-pre-wrap">{{ message().content }}</div>
      }
      @case ('assistant') {
        @if (message().content) {
          <div class="max-w-[85%] rounded-lg border bg-card px-4 py-2 whitespace-pre-wrap">{{ message().content }}</div>
        }
        @for (call of message().tool_calls ?? []; track call.id) {
          <p class="mt-1 text-xs text-muted-foreground">möchte <code>{{ call.name }}</code> aufrufen</p>
        }
      }
      @case ('tool') {
        <details class="max-w-[85%] rounded-md border border-dashed px-3 py-1.5 text-xs">
          <summary class="cursor-pointer">
            <span hlmBadge variant="outline">{{ message().tool_name || 'Werkzeug' }}</span>
            <span class="ml-2 text-muted-foreground">{{ summary() }}</span>
          </summary>
          <pre class="mt-2 max-h-72 overflow-auto">{{ message().content }}</pre>
        </details>
      }
    }
  `,
})
export class MessageView {
  readonly message = input.required<Message>();
  protected readonly summary = computed(() => {
    const text = this.message().content.trim();
    return text.length > 90 ? text.slice(0, 90) + '…' : text;
  });
}
