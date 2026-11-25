/** Types of the sdd-desk HTTP API (internal/desk). */

export interface ToolCall {
  id: string;
  name: string;
  arguments: unknown;
}

export interface Message {
  role: 'user' | 'assistant' | 'tool';
  content: string;
  tool_calls?: ToolCall[];
  tool_call_id?: string;
  tool_name?: string;
}

/** One selectable model. tools is absent when its capability is unknown: a model that cannot call tools is a
 * finding, a model that was not asked is not (desk-model-availability). */
export interface ModelOption {
  name: string;
  tools?: boolean;
  /** Recorded test results per judgment flow (sdd-conflicts, sdd-review); absent means nothing was tested. */
  evidence?: Record<string, EvidenceEntry>;
}

/** A recorded test of a model with a guided flow, from the table in internal/desk/evidence.go. */
export interface EvidenceEntry {
  flow: string;
  protocol: string;
  model_prefix?: string;
  status: 'proven' | 'failed';
  note: string;
  source: string;
}

export interface Provider {
  id: string;
  name: string;
  protocol: string;
  endpoint: string;
  cloud: boolean;
  models: ModelOption[];
  available: boolean;
  note?: string;
}

/** Whether the opened project is covered by sdd (from sdd_stack); missing if it could not be determined. */
export type ProjectState = 'configured' | 'existing' | 'empty';

export interface Meta {
  project: string;
  config_path: string;
  warnings: string[];
  providers: Provider[];
  project_state?: ProjectState;
  /** The guided flows whose answer is the model's judgement; only there is the evidence shown. */
  judgment_flows?: string[];
}

export interface PlannedOp {
  path: string;
  /** create/skip from sdd_new and sdd_init; create/update/unchanged from sdd_write; move from sdd_archive. */
  action: 'create' | 'skip' | 'update' | 'unchanged' | 'move' | 'delete';
  /** Source of a move. */
  from?: string;
  reason?: string;
  content?: string;
  /** Unified diff of an update. */
  diff?: string;
}

export interface Pending {
  call_id: string;
  tool: string;
  arguments: unknown;
  writes: boolean;
  /** A critical action (ADR 0003): runs only after this single confirmation, never per session. */
  critical?: boolean;
  /** A write or critical tool called without apply: shows what would happen, writes nothing (desk-approval-scope). */
  preview_only?: boolean;
  /** The effect in one line; names the action on the confirmation button. */
  summary?: string;
  preview?: {
    applied?: boolean;
    ops?: PlannedOp[];
    error?: string;
    /** sdd_baseline: entries to add and remove, ADRs to lock or re-accept. */
    added?: BaselineEntry[];
    removed?: BaselineEntry[];
    locked?: string[];
    reaccepted?: string[];
    /** sdd_commit: the commit message. */
    message?: string;
    /** sdd_adopt_apply: the adopt report (Markdown). */
    report?: string;
    /** sdd_config: findings that disappear or appear with the new configuration, per check run (ADR 0004). */
    findings_delta?: FindingsDelta[];
    /** sdd_config: check runs that already fail with the current configuration and were not compared. */
    unchecked?: { gate: string; error: string }[];
  };
}

export interface FindingRef {
  rule_id: string;
  path: string;
  line?: number;
  message?: string;
}

export interface FindingsDelta {
  gate: string;
  dropped: FindingRef[];
  added: FindingRef[];
}

export interface BaselineEntry {
  gate: string;
  rule_id: string;
  path: string;
  line?: number;
}

export interface TurnRequest {
  provider: string;
  model: string;
  messages: Message[];
  /** Allowed for the session: reading and previews. */
  allowed_tools: string[];
  /** Allowed for the session: writes, granted only on a card that showed the write (desk-approval-scope). */
  allowed_writes: string[];
  approved_calls: string[];
  denied_calls: string[];
  disclosure_ack: boolean;
}

export interface TurnResponse {
  messages: Message[];
  pending: Pending[];
}

export interface ModelTestRequest {
  provider: string;
  model: string;
  disclosure_ack: boolean;
}

/** Answer of POST /api/model/test; tools is absent if the model did not answer at all. */
export interface ModelTestResult {
  ok: boolean;
  timeout?: boolean;
  tools?: boolean;
  reply?: string;
  error?: string;
  error_plain?: string;
  ms?: number;
}

/** A model test for one provider and model: running, finished with a result, or refused with an error. */
export interface ModelTest {
  provider: string;
  model: string;
  running: boolean;
  result?: ModelTestResult;
  error?: string;
}

/** One line of POST /api/turn/stream. */
export type StreamEvent =
  | { type: 'delta'; text: string }
  | { type: 'thinking'; text: string }
  | { type: 'message'; message: Message }
  | { type: 'done'; messages: Message[]; pending?: Pending[] }
  | { type: 'error'; error: string };

export interface Step {
  id: string;
  title: string;
  state: 'done' | 'open' | 'not-needed' | 'manual' | 'later';
  detail: string;
  command?: string;
}

export interface FeatureGuide {
  feature: string;
  steps: Step[];
  next?: Step;
}

export interface Guide {
  features: FeatureGuide[];
  start?: Step;
  /** Macro step of the BMAD pipeline (ADR 0009): the initiative with its requirements R<n>. */
  initiative?: Step;
  /** Project steps of sdd next, independent of a feature. */
  constitution?: Step;
  adoption?: Step;
  architecture?: Step;
}

export interface PromptArgument {
  name: string;
  description: string;
  required: boolean;
}

export interface PromptInfo {
  name: string;
  title: string;
  description: string;
  arguments: PromptArgument[];
}

/** A decision on a pending tool call. */
export type Decision = 'once' | 'session' | 'deny';

/**
 * Recorded state of one agent CLI (agenten-cli-proxy AC-8). status covers the two ADR 0010 conditions, proxy_proven
 * the run against the desktop's own endpoint; both are separate because a headless contract says nothing about
 * whether the CLI reaches the proxy.
 */
export interface CliEvidence {
  cli: string;
  status: 'proven' | 'failed' | 'unproven';
  proxy_proven: boolean;
  note?: string;
  source: string;
}

/** One agent CLI as the desktop offers it. block_reason names in plain words why it is not offered (AC-1). */
export interface CliInfo {
  id: string;
  name: string;
  cloud: boolean;
  installed: boolean;
  offerable: boolean;
  blocked_because?: string;
  evidence: CliEvidence;
}

export interface CliList {
  cloud_notice: string;
  clis: CliInfo[];
}

/** A tool call of an agent CLI waiting for the user's decision (AC-2, AC-3, AC-10). */
export interface CliCard {
  call_id: string;
  cli: string;
  session: string;
  tool: string;
  tier: 'read' | 'write' | 'critical';
  arguments?: unknown;
  summary?: string;
  deadline: string;
}

/** A decision on a card: once, for the session at the card's tier, or refuse. A session scope never covers a
 * critical call — the backend refuses it with 400 rather than narrowing it silently (AC-2). */
export type CliDecision = 'allow' | 'deny' | 'allow_tool_session' | 'allow_write_session';
