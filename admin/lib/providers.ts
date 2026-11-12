export type ProviderOrigin = "native" | "custom";
export type ProviderStatus = "ok" | "warn";
export type ProviderAccountStatus = "ok" | "warn";
export type ProviderProxyMode = "off" | "preferred" | "required";
export type ProviderProxyHealth = "healthy" | "quarantined" | "unhealthy";
export type ProviderAccountHealth = "active" | "cooldown" | "quarantined" | "unhealthy" | "disabled";

export interface ProviderProxy {
  id: string;
  label: string;
  address: string;
  port: number;
  countryCode: string;
  cityName: string;
  health: ProviderProxyHealth;
}

export interface ProviderAccount {
  id?: string;
  name: string;
  meta: string;
  status: ProviderAccountStatus;
  statusKey: string;
  proxyMode?: ProviderProxyMode;
  proxy?: ProviderProxy | null;
  modelsAvailable?: number;
  modelsUsable?: number;
  capabilities?: string[];
  health?: ProviderAccountHealth;
  active?: boolean;
}

export interface Provider {
  id: string;
  slug?: string;
  catalogSlug?: string;
  name: string;
  icon: string;
  type: string;
  status: ProviderStatus;
  statusKey: string;
  capabilities: string[];
  origins: ProviderOrigin[];
  accounts: ProviderAccount[];
  accountsCount?: number;
  usableAccounts?: number;
  modelsCount?: number;
  modelsUsable?: number;
  signupUrl: string;
  guide: string;
  configured: boolean;
}

const healthy = "common.healthy";

export const PROVIDERS: Record<string, Provider> = {
  openai: {
    id: "openai",
    name: "OpenAI",
    icon: "OA",
    type: "API key · OpenAI",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Vision", "Tools"],
    origins: ["native"],
    configured: true,
    signupUrl: "https://platform.openai.com/api-keys",
    guide: "providers",
    accounts: [
      { name: "produção", meta: "sk-...9f2", status: "ok", statusKey: healthy },
      { name: "staging", meta: "sk-...1a4", status: "ok", statusKey: healthy },
      { name: "sandbox", meta: "sk-...7c1", status: "ok", statusKey: healthy }
    ]
  },
  openrouter: {
    id: "openrouter",
    name: "OpenRouter",
    icon: "OR",
    type: "API key · OpenRouter",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Vision", "Tools"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://openrouter.ai/keys",
    guide: "providers",
    accounts: []
  },
  anthropic: {
    id: "anthropic",
    name: "Anthropic",
    icon: "AN",
    type: "API key · Anthropic",
    status: "warn",
    statusKey: "common.cooldown",
    capabilities: ["Chat", "Tools", "Vision"],
    origins: ["native"],
    configured: true,
    signupUrl: "https://console.anthropic.com/settings/keys",
    guide: "providers",
    accounts: [
      { name: "produção", meta: "sk-ant-...8b2", status: "ok", statusKey: healthy },
      { name: "staging", meta: "sk-ant-...4d9", status: "warn", statusKey: "common.cooldown" }
    ]
  },
  gemini: {
    id: "gemini",
    name: "Gemini",
    icon: "GE",
    type: "API key · Gemini",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Vision", "Tools"],
    origins: ["native"],
    configured: true,
    signupUrl: "https://aistudio.google.com/app/apikey",
    guide: "providers",
    accounts: [{ name: "produção", meta: "AIza...1f3", status: "ok", statusKey: healthy }]
  },
  azure: {
    id: "azure",
    name: "Azure OpenAI",
    icon: "AZ",
    type: "API key · Azure",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Tools"],
    origins: ["native"],
    configured: true,
    signupUrl: "https://portal.azure.com/",
    guide: "providers",
    accounts: [
      { name: "eastus-prod", meta: "az-...7a1", status: "ok", statusKey: healthy },
      { name: "eastus-staging", meta: "az-...2c8", status: "ok", statusKey: healthy }
    ]
  },
  antigravity: {
    id: "antigravity",
    name: "Antigravity",
    icon: "AG",
    type: "OAuth · IDE",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Tools"],
    origins: ["native"],
    configured: true,
    signupUrl: "https://antigravity.google/",
    guide: "providers",
    accounts: [
      { name: "workspace-a", meta: "oauth · ...91", status: "ok", statusKey: healthy },
      { name: "workspace-b", meta: "oauth · ...42", status: "ok", statusKey: healthy },
      { name: "workspace-c", meta: "oauth · ...17", status: "ok", statusKey: healthy },
      { name: "workspace-d", meta: "oauth · ...33", status: "ok", statusKey: healthy },
      { name: "workspace-e", meta: "oauth · ...08", status: "ok", statusKey: healthy }
    ]
  },
  "claude-code": {
    id: "claude-code",
    name: "Claude Code",
    icon: "CC",
    type: "OAuth · Chat",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Tools", "Vision"],
    origins: ["native"],
    configured: true,
    signupUrl: "https://claude.ai/",
    guide: "providers",
    accounts: [{ name: "pessoal", meta: "oauth · ...5a", status: "ok", statusKey: healthy }]
  },
  ollama: {
    id: "ollama",
    name: "Ollama Local",
    icon: "OL",
    type: "Local · sem auth",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Embed"],
    origins: ["native"],
    configured: true,
    signupUrl: "https://ollama.com/download",
    guide: "providers",
    accounts: [{ name: "localhost:11434", meta: "local", status: "ok", statusKey: healthy }]
  },
  "opencode-go": {
    id: "opencode-go",
    name: "OpenCode Go",
    icon: "OG",
    type: "API key · OpenCode Go",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Tools", "Reasoning", "Vision"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://opencode.ai/",
    guide: "providers",
    accounts: []
  },
  "opencode-free": {
    id: "opencode-free",
    name: "OpenCode Free",
    icon: "OC",
    type: "Público · sem auth",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat"],
    origins: ["native"],
    configured: true,
    signupUrl: "https://opencode.ai/",
    guide: "providers",
    accounts: [{ name: "público", meta: "sem credencial", status: "ok", statusKey: healthy }]
  },
  naga: {
    id: "naga",
    name: "Naga AI - EMB",
    icon: "NA",
    type: "Compatível · Embed",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Embed", "Batch"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://naga.ac/",
    guide: "providers",
    accounts: []
  },
  tokenrouter: {
    id: "tokenrouter",
    name: "TokenRouter",
    icon: "TR",
    type: "Compatível · Chat",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Tools"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://tokenrouter.com/",
    guide: "providers",
    accounts: []
  },
  "openai-compatible": {
    id: "openai-compatible",
    name: "OpenAI-compatible",
    icon: "OA",
    type: "Compatível · Chat",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Vision", "Tools"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://platform.openai.com/docs/api-reference",
    guide: "providers",
    accounts: []
  },
  "anthropic-compatible": {
    id: "anthropic-compatible",
    name: "Anthropic-compatible",
    icon: "AN",
    type: "Compatível · Chat",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Tools"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://docs.anthropic.com/",
    guide: "providers",
    accounts: []
  },
  "kilo-code": {
    id: "kilo-code",
    name: "Kilo Code",
    icon: "KC",
    type: "OAuth · Chat",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://kilocode.ai/",
    guide: "providers",
    accounts: []
  },
  "kiro-ai": {
    id: "kiro-ai",
    name: "Kiro AI",
    icon: "KI",
    type: "OAuth · Chat",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Tools"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://kiro.dev/",
    guide: "providers",
    accounts: []
  },
  "openai-codex": {
    id: "openai-codex",
    name: "OpenAI Codex",
    icon: "CX",
    type: "OAuth · Chat",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Tools"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://openai.com/codex/",
    guide: "providers",
    accounts: []
  },
  "gemini-cli": {
    id: "gemini-cli",
    name: "Gemini CLI",
    icon: "GC",
    type: "OAuth · Chat",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Vision"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://github.com/google-gemini/gemini-cli",
    guide: "providers",
    accounts: []
  },
  bedrock: {
    id: "bedrock",
    name: "Amazon Bedrock",
    icon: "BR",
    type: "API key · Bedrock",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Tools"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://aws.amazon.com/bedrock/",
    guide: "providers",
    accounts: []
  },
  groq: {
    id: "groq",
    name: "Groq",
    icon: "GR",
    type: "API key · Groq",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat", "Tools"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://console.groq.com/keys",
    guide: "providers",
    accounts: []
  },
  "lm-studio": {
    id: "lm-studio",
    name: "LM Studio",
    icon: "LM",
    type: "Local · sem auth",
    status: "ok",
    statusKey: healthy,
    capabilities: ["Chat"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://lmstudio.ai/",
    guide: "providers",
    accounts: []
  },
  agentrouter: {
    id: "agentrouter",
    name: "AgentRouter",
    icon: "AR",
    type: "Anthropic · Chat",
    status: "ok",
    statusKey: "providers.available",
    capabilities: ["Chat", "Tools"],
    origins: ["native"],
    configured: false,
    signupUrl: "https://agentrouter.org/",
    guide: "providers",
    accounts: []
  }
};

// UI-only signing pages for catalog slugs that have no static entry in PROVIDERS.
export const SIGNUP_URLS: Record<string, string> = {
  alibaba: "https://dashscope.console.aliyun.com/apiKey",
  cerebras: "https://cloud.cerebras.ai/",
  cohere: "https://dashboard.cohere.com/api-keys",
  mistral: "https://console.mistral.ai/api-keys",
  nvidia: "https://build.nvidia.com/",
  perplexity: "https://www.perplexity.ai/settings/api",
  together: "https://api.together.ai/settings/api-keys",
  venice: "https://venice.ai/settings/api",
  "vercel-ai-gateway": "https://vercel.com/dashboard/ai-gateway",
  xai: "https://console.x.ai/"
};

export const CONFIGURED_PROVIDERS = Object.values(PROVIDERS).filter(provider => provider.configured);
export const CATALOG_PROVIDERS = Object.values(PROVIDERS).filter(provider => !provider.configured);
export const ORIGIN_ORDER: ProviderOrigin[] = ["native", "custom"];
