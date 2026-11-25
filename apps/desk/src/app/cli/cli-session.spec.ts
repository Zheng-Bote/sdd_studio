import { describe, expect, it } from 'vitest';
import { CliCard, CliInfo } from '../core/types';
import { cliLabel, cliStatusText, secondsLeft, sessionDecision, tierText } from './cli-session';

function info(patch: Partial<CliInfo> = {}): CliInfo {
  return {
    id: 'claude',
    name: 'Claude Code',
    cloud: true,
    installed: true,
    offerable: true,
    evidence: { cli: 'claude', status: 'proven', proxy_proven: true, source: 'test' },
    ...patch,
  };
}

function card(patch: Partial<CliCard> = {}): CliCard {
  return { call_id: 'c-1', cli: 'claude', session: 's-abc12345', tool: 'sdd_next', tier: 'read', deadline: '', ...patch };
}

describe('agenten-cli-proxy AC-1: the selection names what is offered and why not', () => {
  it('shows the plain name for an offered CLI', () => {
    expect(cliLabel(info())).toBe('Claude Code');
  });

  it('names the reason a CLI is not offered', () => {
    expect(cliLabel(info({ offerable: false, blocked_because: 'nicht installiert' }))).toBe(
      'Claude Code — nicht angeboten: nicht installiert',
    );
  });
});

describe('agenten-cli-proxy AC-8: a CLI without a record is never called proven', () => {
  it('calls a proven CLI belegt', () => {
    expect(cliStatusText(info())).toBe('belegt');
  });

  it('calls a CLI without the proxy run nicht belegt', () => {
    expect(cliStatusText(info({ evidence: { cli: 'claude', status: 'proven', proxy_proven: false, source: 'x' } }))).toBe(
      'nicht belegt',
    );
  });

  it('calls an unproven CLI nicht belegt', () => {
    expect(cliStatusText(info({ evidence: { cli: 'agy', status: 'unproven', proxy_proven: false, source: 'x' } }))).toBe(
      'nicht belegt',
    );
  });
});

describe('agenten-cli-proxy AC-2: a session allowance is never offered for a critical call', () => {
  it('offers no session scope for a critical card', () => {
    expect(sessionDecision(card({ tier: 'critical' }))).toBeNull();
  });

  it('offers the read scope for a reading card and the write scope for a writing one', () => {
    expect(sessionDecision(card({ tier: 'read' }))).toBe('allow_tool_session');
    expect(sessionDecision(card({ tier: 'write' }))).toBe('allow_write_session');
  });
});

describe('agenten-cli-proxy AC-3: a card names its tier, and the tier is read in words', () => {
  it('names the tier in words', () => {
    expect(tierText('read')).toBe('lesend');
    expect(tierText('write')).toBe('schreibend');
    expect(tierText('critical')).toBe('kritisch');
  });
});

describe('agenten-cli-proxy AC-4: an expired card reports no time left', () => {
  it('counts down to the deadline', () => {
    const now = Date.parse('2026-11-25T10:00:00Z');
    expect(secondsLeft(card({ deadline: '2026-11-25T10:00:30Z' }), now)).toBe(30);
  });

  it('reports 0 once the deadline passed', () => {
    const now = Date.parse('2026-11-25T10:01:00Z');
    expect(secondsLeft(card({ deadline: '2026-11-25T10:00:30Z' }), now)).toBe(0);
  });
});
