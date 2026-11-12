export type ProviderAuthType = "api_key" | "oauth" | "none";
export type ProviderSetupField = "credential" | "base_url" | "model_id";

export interface ProviderSetup {
  authType: ProviderAuthType;
  requiredFields: ProviderSetupField[];
  kind?: "openai" | "anthropic" | "agentrouter";
  defaultBaseUrl?: string;
  defaultModel?: string;
  supported: boolean;
}

export const PROVIDER_SETUP: Record<string, ProviderSetup> = {
  openai: {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    kind: "openai",
    defaultBaseUrl: "https://api.openai.com/v1",
    defaultModel: "gpt-4.1",
    supported: true
  },
  openrouter: {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    kind: "openai",
    defaultBaseUrl: "https://openrouter.ai/api/v1",
    defaultModel: "openai/gpt-4o-mini",
    supported: true
  },
  anthropic: {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    kind: "anthropic",
    defaultBaseUrl: "https://api.anthropic.com",
    defaultModel: "claude-sonnet-4",
    supported: true
  },
  gemini: {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    supported: false
  },
  azure: {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    supported: false
  },
  antigravity: {
    authType: "oauth",
    requiredFields: [],
    supported: false
  },
  "claude-code": {
    authType: "oauth",
    requiredFields: [],
    supported: false
  },
  ollama: {
    authType: "none",
    requiredFields: ["base_url", "model_id"],
    supported: false
  },
  "opencode-go": {
    authType: "api_key",
    requiredFields: ["credential"],
    kind: "openai",
    defaultBaseUrl: "https://opencode.ai/inference/go/openai/v1",
    defaultModel: "deepseek-v4.1-flash",
    supported: true
  },
  "opencode-free": {
    authType: "none",
    requiredFields: [],
    supported: false
  },
  naga: {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    supported: false
  },
  tokenrouter: {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    supported: false
  },
  "openai-compatible": {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    kind: "openai",
    supported: true
  },
  "anthropic-compatible": {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    kind: "anthropic",
    defaultBaseUrl: "https://api.anthropic.com",
    defaultModel: "claude-sonnet-4",
    supported: true
  },
  "kilo-code": {
    authType: "oauth",
    requiredFields: [],
    supported: false
  },
  "kiro-ai": {
    authType: "oauth",
    requiredFields: [],
    supported: false
  },
  "openai-codex": {
    authType: "oauth",
    requiredFields: [],
    supported: false
  },
  "gemini-cli": {
    authType: "oauth",
    requiredFields: [],
    supported: false
  },
  bedrock: {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    supported: false
  },
  groq: {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    kind: "openai",
    defaultBaseUrl: "https://api.groq.com/openai/v1",
    defaultModel: "llama-3.3-70b-versatile",
    supported: true
  },
  "lm-studio": {
    authType: "none",
    requiredFields: ["base_url", "model_id"],
    supported: false
  },
  agentrouter: {
    authType: "api_key",
    requiredFields: ["credential", "base_url", "model_id"],
    kind: "agentrouter",
    defaultBaseUrl: "https://agentrouter.org",
    defaultModel: "gpt-5.6-sol",
    supported: true
  }
};
