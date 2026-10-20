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

export interface Meta {
  project: string;
  config_path: string;
  warnings: string[];
  providers: Provider[];
}

export interface PlannedOp {
  path: string;
  action: 'create' | 'skip';
  reason?: string;
  content?: string;
}

export interface Pending {
  call_id: string;
  tool: string;
  arguments: unknown;
  writes: boolean;
  preview?: { applied?: boolean; ops?: PlannedOp[]; error?: string };
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
