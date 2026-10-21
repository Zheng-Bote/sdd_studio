import { TestBed } from '@angular/core/testing';
import { ApiError, DeskApi, NdjsonLines } from './desk-api';
import { ChatSession, trimOpenCalls } from './chat-session';
import { modelTestText } from '../app';
import { Message, Meta, ModelTestRequest, ModelTestResult, StreamEvent, TurnRequest, TurnResponse } from './types';

const meta: Meta = {
  project: 'demo',
  config_path: 'x',
  warnings: [],
  providers: [
    { id: 'ollama-local', name: 'Ollama (lokal)', protocol: 'ollama', endpoint: '', cloud: false, models: ['qwen3:14b'], available: true },
    { id: 'claude', name: 'Claude', protocol: 'claude', endpoint: '', cloud: true, models: ['claude-x'], available: true },
  ],
};

/** A turn's events; a plain response becomes a single done event. */
type Script = TurnResponse | StreamEvent[];

function setup(responses: Script[], tests: (ModelTestResult | ApiError)[] = []) {
  const requests: TurnRequest[] = [];
  const testRequests: ModelTestRequest[] = [];
  const api = {
    meta: async () => meta,
    testModel: async (req: ModelTestRequest) => {
      testRequests.push(structuredClone(req));
      const res = tests.shift();
      if (!res || res instanceof ApiError) throw res ?? new ApiError('no more tests', 500);
      return res;
    },
    turnStream: async (req: TurnRequest, onEvent: (e: StreamEvent) => void) => {
      requests.push(structuredClone(req));
      const res = responses.shift();
      if (!res) throw new ApiError('no more responses', 500);
      const events: StreamEvent[] = Array.isArray(res) ? res : [{ type: 'done', ...res }];
      events.forEach(onEvent);
    },
  };
  TestBed.configureTestingModule({ providers: [{ provide: DeskApi, useValue: api }] });
  return { session: TestBed.inject(ChatSession), requests, testRequests };
}

const pendingRead = { call_id: 'c1', tool: 'sdd_next', arguments: {}, writes: false };
const pendingWrite = { call_id: 'w1', tool: 'sdd_new', arguments: { apply: true }, writes: true };

describe('ChatSession', () => {
  // sdd-desk AC-5: a cloud provider needs the disclosure before anything is sent.
  it('blocks sending to a cloud provider until the disclosure is acknowledged', async () => {
    const { session, requests } = setup([{ messages: [{ role: 'assistant', content: 'ok' }], pending: [] }]);
    await session.load();
    expect(session.providerId()).toBe('ollama-local');
    expect(session.needsDisclosure()).toBe(false);

    session.selectProvider('claude');
    expect(session.needsDisclosure()).toBe(true);
    expect(session.canSend()).toBe(false);
    await session.send('Hallo');
    expect(requests.length).toBe(0);

    session.acknowledge();
    await session.send('Hallo');
    expect(requests[0].disclosure_ack).toBe(true);
    expect(requests[0].model).toBe('claude-x');
  });

  // sdd-desk AC-4: once, session and deny reach the backend; the turn continues only when every call is decided.
  it('collects decisions and continues after the last one', async () => {
    const { session, requests } = setup([
      { messages: [{ role: 'user', content: 'x' }], pending: [pendingRead, { ...pendingRead, call_id: 'c2', tool: 'sdd_lint' }] },
      { messages: [{ role: 'assistant', content: 'fertig' }], pending: [] },
    ]);
    await session.load();
    await session.send('x');
    expect(session.pending().length).toBe(2);

    await session.decide('c1', 'session');
    expect(requests.length).toBe(1); // still waiting for c2
    await session.decide('c2', 'deny');

    expect(requests.length).toBe(2);
    expect(requests[1].approved_calls).toEqual(['c1']);
    expect(requests[1].denied_calls).toEqual(['c2']);
    expect(requests[1].allowed_tools).toEqual(['sdd_next']);
    expect(session.pending().length).toBe(0);
  });

  // sdd-desk AC-4: a write is never allowed for the whole session.
  it('downgrades a session approval of a write to a one-time approval', async () => {
    const { session, requests } = setup([
      { messages: [], pending: [pendingWrite] },
      { messages: [], pending: [] },
    ]);
    await session.load();
    await session.send('anlegen');
    await session.decide('w1', 'session');
    expect(requests[1].approved_calls).toEqual(['w1']);
    expect(requests[1].allowed_tools).toEqual([]);
  });

  it('withdraws the acknowledgement when the backend demands the disclosure again', async () => {
    const { session } = setup([]);
    const api = TestBed.inject(DeskApi) as unknown as { turnStream: () => Promise<never> };
    api.turnStream = async () => {
      throw new ApiError('Hinweis nicht bestätigt', 428);
    };
    await session.load();
    session.selectProvider('claude');
    session.acknowledge();
    await session.send('x');
    expect(session.needsDisclosure()).toBe(true);
    expect(session.error()).toContain('Hinweis');
  });

  // sdd-desk-streaming AC-6: the reasoning is shown while the model works and is dropped afterwards.
  // sdd-desk-streaming AC-5: reasoning and answer grow while the model works; reasoning never enters the conversation.
  it('shows the streamed text and then the finished messages', async () => {
    const seen: string[] = [];
    const answer: Message = { role: 'assistant', content: 'Hallo Welt' };
    const { session } = setup([]);
    const api = TestBed.inject(DeskApi) as unknown as { turnStream: DeskApi['turnStream'] };
    api.turnStream = async (_req, onEvent) => {
      onEvent({ type: 'thinking', text: 'Erst überlegen.' });
      seen.push(session.thinking());
      onEvent({ type: 'delta', text: 'Hallo' });
      seen.push(session.streamingMessage()?.content ?? '');
      onEvent({ type: 'delta', text: ' Welt' });
      seen.push(session.streamingMessage()?.content ?? '');
      onEvent({ type: 'message', message: answer });
      seen.push(session.streaming());
      onEvent({ type: 'done', messages: [{ role: 'user', content: 'x' }, answer] });
    };
    await session.load();
    expect(session.canSend()).toBe(true);
    const turn = session.send('x');
    expect(session.busy()).toBe(true);
    expect(session.canSend()).toBe(false);
    await turn;
    expect(seen).toEqual(['Erst überlegen.', 'Hallo', 'Hallo Welt', '']);
    expect(session.thinking()).toBe('');
    expect(session.messages().some((m) => m.content.includes('überlegen'))).toBe(false);
    expect(session.messages().at(-1)).toEqual(answer);
    expect(session.pending()).toEqual([]);
    expect(session.busy()).toBe(false);
  });

  // sdd-desk-streaming AC-4: after cancel the finished messages stay, the partial text is kept and an open call is dropped.
  it('keeps finished messages and drops an unanswered call when the turn is cancelled', async () => {
    const { session } = setup([]);
    const api = TestBed.inject(DeskApi) as unknown as { turnStream: DeskApi['turnStream'] };
    api.turnStream = (_req, onEvent, signal) =>
      new Promise((_resolve, reject) => {
        onEvent({ type: 'message', message: { role: 'assistant', content: '', tool_calls: [{ id: 'c1', name: 'sdd_next', arguments: {} }] } });
        onEvent({ type: 'message', message: { role: 'tool', content: 'ok', tool_call_id: 'c1', tool_name: 'sdd_next' } });
        onEvent({ type: 'delta', text: 'Der nächste Schritt' });
        signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')));
      });
    await session.load();
    const turn = session.send('Was nun?');
    session.cancel();
    await turn;
    expect(session.messages().map((m) => m.role)).toEqual(['user', 'assistant', 'tool', 'assistant']);
    expect(session.messages().at(-1)?.content).toBe('Der nächste Schritt');
    expect(session.error()).toBe('Abgebrochen.');
    expect(session.busy()).toBe(false);
    expect(session.canSend()).toBe(true);
  });

  // sdd-desk-streaming AC-4: an error event mid-turn ends it like an abort, with the error shown.
  it('ends the turn on an error event', async () => {
    const { session } = setup([
      [
        { type: 'message', message: { role: 'assistant', content: '', tool_calls: [{ id: 'c9', name: 'sdd_lint', arguments: {} }] } },
        { type: 'error', error: 'Claude: HTTP 529: overloaded' },
      ],
    ]);
    await session.load();
    await session.send('prüfen');
    expect(session.messages().map((m) => m.role)).toEqual(['user']);
    expect(session.error()).toContain('overloaded');
  });
});

describe('model test', () => {
  // sdd-desk-model-test AC-5: the test needs a model and the disclosure, and its result belongs to exactly that model.
  it('tests the selected model and drops the result when the selection changes', async () => {
    const { session, testRequests } = setup([], [
      { ok: true, tools: true, reply: 'Hello World', ms: 120 },
      { ok: true, tools: true, reply: 'spät', ms: 5 },
    ]);
    await session.load();
    const running = session.testModel();
    expect(session.modelTest()?.running).toBe(true);
    expect(session.canTest()).toBe(false);
    await running;
    expect(testRequests).toEqual([{ provider: 'ollama-local', model: 'qwen3:14b', disclosure_ack: false }]);
    expect(modelTestText(session.modelTest()!)).toContain('Werkzeuge ok');

    session.selectModel('anderes');
    expect(session.modelTest()).toBeNull();
    const late = session.testModel();
    session.selectModel('qwen3:14b');
    await late;
    expect(session.modelTest()).toBeNull(); // a result for a model no longer selected is dropped
    session.selectProvider('claude');
    expect(session.canTest()).toBe(false);
    await session.testModel();
    expect(testRequests.length).toBe(2);
  });

  // sdd-desk-model-test AC-5: results without tools, without answer, timeouts and refusals are told apart; 428 revokes the disclosure.
  it('shows every outcome', async () => {
    const { session, testRequests } = setup(
      [],
      [
        { ok: true, tools: false, reply: 'Hi', error: 'HTTP 404: no tool use', ms: 80 },
        { ok: false, error: 'HTTP 404: No endpoints', error_plain: 'HTTP 404: No endpoints' },
        { ok: false, timeout: true, error: 'context deadline exceeded' },
        new ApiError('Hinweis nicht bestätigt', 428),
      ],
    );
    await session.load();
    session.selectProvider('claude');
    session.acknowledge();
    await session.testModel();
    expect(testRequests[0].disclosure_ack).toBe(true);
    expect(modelTestText(session.modelTest()!)).toContain('ohne Werkzeuge');
    await session.testModel();
    expect(modelTestText(session.modelTest()!)).toContain('antwortet nicht: HTTP 404');
    await session.testModel();
    expect(modelTestText(session.modelTest()!)).toContain('keine Antwort innerhalb');
    await session.testModel();
    expect(modelTestText(session.modelTest()!)).toContain('abgelehnt');
    expect(session.needsDisclosure()).toBe(true);
  });
});

describe('trimOpenCalls', () => {
  // sdd-desk-streaming AC-4: only an assistant message whose calls lack results is removed, with everything after it.
  it('removes an assistant message with unanswered calls', () => {
    const call = (id: string): Message => ({ role: 'assistant', content: '', tool_calls: [{ id, name: 'sdd_next', arguments: {} }] });
    const result = (id: string): Message => ({ role: 'tool', content: 'ok', tool_call_id: id });
    const user: Message = { role: 'user', content: 'x' };
    expect(trimOpenCalls([user, call('a'), result('a')])).toEqual([user, call('a'), result('a')]);
    expect(trimOpenCalls([user, call('a')])).toEqual([user]);
    expect(trimOpenCalls([user, call('a'), result('a'), call('b')])).toEqual([user, call('a'), result('a')]);
    expect(trimOpenCalls([user])).toEqual([user]);
  });
});

describe('NdjsonLines', () => {
  // sdd-desk-streaming AC-2: events split across chunks are put back together line by line.
  it('reassembles events from arbitrary chunks', () => {
    const lines = new NdjsonLines();
    expect(lines.push('{"type":"delta","te')).toEqual([]);
    expect(lines.push('xt":"Hal"}\n{"type":"delta","text":"lo"}\n\n{"type":')).toEqual([
      { type: 'delta', text: 'Hal' },
      { type: 'delta', text: 'lo' },
    ]);
    expect(lines.push('"x"}')).toEqual([]);
    expect(lines.end()).toEqual([{ type: 'x' }]);
    expect(lines.end()).toEqual([]);
  });
});
