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

export interface Provider {
  id: string;
  name: string;
  protocol: string;
  endpoint: string;
  cloud: boolean;
  models: string[];
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
}

export interface PlannedOp {
  path: string;
  /** create/skip from sdd_new and sdd_init; create/update/unchanged from sdd_write; move from sdd_archive. */
  action: 'create' | 'skip' | 'update' | 'unchanged' | 'move';
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
    /** sdd_adopt_apply: the adopt report (Markdown). */
    report?: string;
  };
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
  allowed_tools: string[];
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
