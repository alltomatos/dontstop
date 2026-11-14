// Package catalog defines the MCP tool catalog shared by the admin console and
// the MCP runtime. It has no dependencies on services or transport so both
// sides can use it without import cycles.
package catalog

// Tool describes one MCP tool and its security/annotation metadata.
type Tool struct {
	ID          string
	Name        string
	Scope       string
	Description string
	ReadOnly    bool
	Destructive bool
	Idempotent  bool
	OpenWorld   bool
}

// Tools is the canonical, deterministic tool catalog.
var Tools = []Tool{
	{
		ID: "dontstop_chat", Name: "dontstop_chat", Scope: "chat:write",
		Description: "Run a non-streaming chat completion through the gateway routing, cache and failover.",
	},
	{
		ID: "dontstop_models", Name: "dontstop_models", Scope: "models:read", ReadOnly: true,
		Description: "List active models available to the tenant.",
	},
	{
		ID: "dontstop_embeddings", Name: "dontstop_embeddings", Scope: "embeddings:write", ReadOnly: true,
		Description: "Create embeddings. Returns an actionable error while no embeddings backend is configured.",
	},
	{
		ID: "dontstop_web_search", Name: "dontstop_web_search", Scope: "web:read", ReadOnly: true,
		Description: "Search the web. Returns an actionable error while no web search backend is configured.",
		OpenWorld: true,
	},
	{
		ID: "dontstop_usage", Name: "dontstop_usage", Scope: "usage:read", ReadOnly: true,
		Description: "Read aggregated usage and cost for the tenant.",
	},
	{
		ID: "dontstop_health", Name: "dontstop_health", ReadOnly: true,
		Description: "Report MCP runtime liveness without exposing secrets.",
	},
	{
		ID: "dontstop_combo_catalog", Name: "dontstop_combo_catalog", Scope: "combos:read", ReadOnly: true,
		Description: "List strategies and provider models eligible for combos.",
	},
	{
		ID: "dontstop_combo_list", Name: "dontstop_combo_list", Scope: "combos:read", ReadOnly: true,
		Description: "List tenant combos with member and strategy summary.",
	},
	{
		ID: "dontstop_combo_get", Name: "dontstop_combo_get", Scope: "combos:read", ReadOnly: true,
		Description: "Get one combo with its members and status.",
	},
	{
		ID: "dontstop_combo_create", Name: "dontstop_combo_create", Scope: "combos:write",
		Description: "Create a combo after validating its provider/model members.",
	},
	{
		ID: "dontstop_combo_update", Name: "dontstop_combo_update", Scope: "combos:write",
		Description: "Update a combo metadata/members using optimistic revision.",
	},
	{
		ID: "dontstop_combo_activate", Name: "dontstop_combo_activate", Scope: "combos:write", Idempotent: true,
		Description: "Activate a combo that has at least one usable member.",
	},
	{
		ID: "dontstop_combo_deactivate", Name: "dontstop_combo_deactivate", Scope: "combos:write", Idempotent: true,
		Description: "Deactivate a combo without deleting it.",
	},
	{
		ID: "dontstop_combo_delete", Name: "dontstop_combo_delete", Scope: "combos:write", Destructive: true,
		Description: "Delete a combo and its members.",
	},
	{
		ID: "dontstop_provider_list", Name: "dontstop_provider_list", Scope: "models:read", ReadOnly: true,
		Description: "List connected providers and models with capabilities.",
	},
}

// Find returns the tool with the given ID.
func Find(id string) (Tool, bool) {
	for _, tool := range Tools {
		if tool.ID == id {
			return tool, true
		}
	}
	return Tool{}, false
}
