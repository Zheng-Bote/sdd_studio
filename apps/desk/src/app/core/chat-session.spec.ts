import { TestBed } from '@angular/core/testing';
import { ApiError, DeskApi } from './desk-api';
import { ChatSession } from './chat-session';
import { Meta, TurnRequest, TurnResponse } from './types';

const meta: Meta = {
  project: 'demo',
  config_path: 'x',
  warnings: [],
  providers: [
    { id: 'ollama-local', name: 'Ollama (lokal)', protocol: 'ollama', endpoint: '', cloud: false, models: ['qwen3:14b'], available: true },
    { id: 'claude', name: 'Claude', protocol: 'claude', endpoint: '', cloud: true, models: ['claude-x'], available: true },
  ],
};

function setup(responses: TurnResponse[]) {
  const requests: TurnRequest[] = [];
  const api = {
    meta: async () => meta,
    turn: async (req: TurnRequest) => {
      requests.push(structuredClone(req));
      const res = responses.shift();
      if (!res) throw new ApiError('no more responses', 500);
      return res;
    },
  };
  TestBed.configureTestingModule({ providers: [{ provide: DeskApi, useValue: api }] });
  return { session: TestBed.inject(ChatSession), requests };
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
    const api = TestBed.inject(DeskApi) as unknown as { turn: () => Promise<never> };
    api.turn = async () => {
      throw new ApiError('Hinweis nicht bestätigt', 428);
    };
    await session.load();
    session.selectProvider('claude');
    session.acknowledge();
    await session.send('x');
    expect(session.needsDisclosure()).toBe(true);
    expect(session.error()).toContain('Hinweis');
  });
});
