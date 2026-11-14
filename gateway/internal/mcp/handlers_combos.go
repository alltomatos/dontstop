package mcp

import (
	"context"
	"errors"

	"github.com/alltomatos/dontstop/gateway/internal/admin"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type comboMemberInput struct {
	ProviderID  string         `json:"provider_id" jsonschema:"provider ID"`
	ModelID     string         `json:"model_id" jsonschema:"model ID"`
	Priority    int            `json:"priority,omitempty" jsonschema:"priority (lower first)"`
	Position    int            `json:"position,omitempty" jsonschema:"position inside priority"`
	Weight      int            `json:"weight,omitempty" jsonschema:"routing weight"`
	Enabled     *bool          `json:"enabled,omitempty" jsonschema:"member enabled"`
	Constraints map[string]any `json:"constraints,omitempty" jsonschema:"optional constraints"`
}

type comboInput struct {
	Slug            string             `json:"slug" jsonschema:"combo slug"`
	Name            string             `json:"name" jsonschema:"combo name"`
	Description     string             `json:"description,omitempty" jsonschema:"combo description"`
	Strategy        string             `json:"strategy" jsonschema:"priority, round_robin or priority_round_robin"`
	FailoverEnabled bool               `json:"failover_enabled,omitempty" jsonschema:"enable failover"`
	RPMLimit        *int               `json:"rpm_limit,omitempty" jsonschema:"optional RPM limit"`
	TPMLimit        *int               `json:"tpm_limit,omitempty" jsonschema:"optional TPM limit"`
	MaxAttempts     int                `json:"max_attempts,omitempty" jsonschema:"max attempts (default 5)"`
	Source          string             `json:"source,omitempty" jsonschema:"manual or ai"`
	Members         []comboMemberInput `json:"members" jsonschema:"combo members"`
}

type comboUpdateInput struct {
	ID       string     `json:"id" jsonschema:"combo ID"`
	Revision int        `json:"revision" jsonschema:"current combo revision"`
	Combo    comboInput `json:"combo" jsonschema:"full combo definition"`
}

func (input comboInput) toAdmin() admin.ComboInput {
	allPositionsZero := true
	for _, member := range input.Members {
		if member.Position != 0 {
			allPositionsZero = false
			break
		}
	}
	members := make([]admin.ComboMemberInput, len(input.Members))
	for index, member := range input.Members {
		enabled := true
		if member.Enabled != nil {
			enabled = *member.Enabled
		}
		position := member.Position
		if allPositionsZero {
			position = index
		}
		weight := member.Weight
		if weight == 0 {
			weight = 1
		}
		members[index] = admin.ComboMemberInput{
			ProviderID: member.ProviderID, ModelID: member.ModelID,
			Priority: member.Priority, Position: position, Weight: weight,
			Enabled: enabled, Constraints: member.Constraints,
		}
	}
	return admin.ComboInput{
		Slug: input.Slug, Name: input.Name, Description: input.Description,
		Strategy: input.Strategy, FailoverEnabled: input.FailoverEnabled,
		RPMLimit: input.RPMLimit, TPMLimit: input.TPMLimit,
		MaxAttempts: input.MaxAttempts, Source: input.Source, Members: members,
	}
}

func (s *Service) comboCatalogHandler(tenantID string) sdkmcp.ToolHandlerFor[emptyInput, any] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ emptyInput) (*sdkmcp.CallToolResult, any, error) {
		if s.combos == nil {
			return nil, nil, errors.New("combo service is not configured")
		}
		models, err := s.combos.Catalog(ctx, tenantID)
		if err != nil {
			s.logger.Warn("mcp combo catalog failed", "tenant_id", tenantID, "error", err)
			return nil, nil, errors.New("failed to load combo catalog")
		}
		return toolResult(models)
	}
}

func (s *Service) comboListHandler(tenantID string) sdkmcp.ToolHandlerFor[emptyInput, any] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ emptyInput) (*sdkmcp.CallToolResult, any, error) {
		if s.combos == nil {
			return nil, nil, errors.New("combo service is not configured")
		}
		combos, err := s.combos.List(ctx, tenantID)
		if err != nil {
			s.logger.Warn("mcp combo list failed", "tenant_id", tenantID, "error", err)
			return nil, nil, errors.New("failed to list combos")
		}
		return toolResult(combos)
	}
}

func (s *Service) comboGetHandler(tenantID string) sdkmcp.ToolHandlerFor[toolIDInput, any] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, input toolIDInput) (*sdkmcp.CallToolResult, any, error) {
		if s.combos == nil {
			return nil, nil, errors.New("combo service is not configured")
		}
		combo, err := s.combos.Get(ctx, tenantID, input.ID)
		if err != nil {
			return nil, nil, comboError(err)
		}
		return toolResult(combo)
	}
}

func (s *Service) comboCreateHandler(tenantID, actor string) sdkmcp.ToolHandlerFor[comboInput, any] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, input comboInput) (*sdkmcp.CallToolResult, any, error) {
		if s.combos == nil {
			return nil, nil, errors.New("combo service is not configured")
		}
		combo, err := s.combos.Create(ctx, tenantID, input.toAdmin())
		if err != nil {
			s.auditCombo(ctx, tenantID, actor, "mcp.combo.create", input.Slug, err)
			return nil, nil, comboError(err)
		}
		s.auditCombo(ctx, tenantID, actor, "mcp.combo.create", combo.ID, nil)
		return toolResult(combo)
	}
}

func (s *Service) comboUpdateHandler(tenantID, actor string) sdkmcp.ToolHandlerFor[comboUpdateInput, any] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, input comboUpdateInput) (*sdkmcp.CallToolResult, any, error) {
		if s.combos == nil {
			return nil, nil, errors.New("combo service is not configured")
		}
		combo, err := s.combos.Update(ctx, tenantID, input.ID, input.Revision, input.Combo.toAdmin())
		if err != nil {
			s.auditCombo(ctx, tenantID, actor, "mcp.combo.update", input.ID, err)
			return nil, nil, comboError(err)
		}
		s.auditCombo(ctx, tenantID, actor, "mcp.combo.update", combo.ID, nil)
		return toolResult(combo)
	}
}

func (s *Service) comboStatusHandler(tenantID, actor, status string) sdkmcp.ToolHandlerFor[toolIDInput, any] {
	event := "mcp.combo.activate"
	if status == "inactive" {
		event = "mcp.combo.deactivate"
	}
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, input toolIDInput) (*sdkmcp.CallToolResult, any, error) {
		if s.combos == nil {
			return nil, nil, errors.New("combo service is not configured")
		}
		combo, err := s.combos.SetStatus(ctx, tenantID, input.ID, status)
		if err != nil {
			s.auditCombo(ctx, tenantID, actor, event, input.ID, err)
			return nil, nil, comboError(err)
		}
		s.auditCombo(ctx, tenantID, actor, event, combo.ID, nil)
		return toolResult(combo)
	}
}

func (s *Service) comboDeleteHandler(tenantID, actor string) sdkmcp.ToolHandlerFor[toolIDInput, any] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, input toolIDInput) (*sdkmcp.CallToolResult, any, error) {
		if s.combos == nil {
			return nil, nil, errors.New("combo service is not configured")
		}
		if err := s.combos.Delete(ctx, tenantID, input.ID); err != nil {
			s.auditCombo(ctx, tenantID, actor, "mcp.combo.delete", input.ID, err)
			return nil, nil, comboError(err)
		}
		s.auditCombo(ctx, tenantID, actor, "mcp.combo.delete", input.ID, nil)
		return toolResult(map[string]bool{"deleted": true})
	}
}

func (s *Service) providerListHandler(tenantID string) sdkmcp.ToolHandlerFor[emptyInput, any] {
	return func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ emptyInput) (*sdkmcp.CallToolResult, any, error) {
		if s.combos == nil {
			return nil, nil, errors.New("provider catalog is not configured")
		}
		models, err := s.combos.Catalog(ctx, tenantID)
		if err != nil {
			s.logger.Warn("mcp provider list failed", "tenant_id", tenantID, "error", err)
			return nil, nil, errors.New("failed to list providers")
		}
		return toolResult(models)
	}
}

func (s *Service) auditCombo(ctx context.Context, tenantID, actor, event, comboID string, err error) {
	if s.store == nil {
		return
	}
	payload := map[string]any{"combo": comboID}
	if err != nil {
		payload["status"] = "error"
	} else {
		payload["status"] = "ok"
	}
	if auditErr := s.store.RecordAudit(ctx, tenantID, actor, event, payload); auditErr != nil {
		s.logger.Warn("mcp combo audit failed", "event", event, "error", auditErr)
	}
}

func comboError(err error) error {
	switch {
	case errors.Is(err, admin.ErrNotFound):
		return errors.New("combo not found")
	case errors.Is(err, admin.ErrInvalidInput):
		return errors.New("invalid combo input")
	case errors.Is(err, admin.ErrConflict):
		return errors.New("combo conflict or no usable member")
	default:
		return errors.New("combo operation failed")
	}
}
