package routing

import "testing"

func TestRotateTargetsRoundRobin(t *testing.T) {
	targets := []ComboTarget{{ModelID: "a"}, {ModelID: "b"}, {ModelID: "c"}}
	got := rotateTargets(targets, "round_robin", 1)
	if got[0].ModelID != "b" || got[1].ModelID != "c" || got[2].ModelID != "a" {
		t.Fatalf("unexpected rotation: %+v", got)
	}
}

func TestRotateTargetsPriorityRoundRobinKeepsTiers(t *testing.T) {
	targets := []ComboTarget{
		{ModelID: "a", Priority: 0},
		{ModelID: "b", Priority: 0},
		{ModelID: "c", Priority: 1},
	}
	got := rotateTargets(targets, "priority_round_robin", 1)
	if got[0].ModelID != "b" || got[1].ModelID != "a" || got[2].ModelID != "c" {
		t.Fatalf("unexpected tier rotation: %+v", got)
	}
}

func TestRotateTargetsPriorityIsStable(t *testing.T) {
	targets := []ComboTarget{{ModelID: "a", Priority: 0}, {ModelID: "b", Priority: 1}}
	got := rotateTargets(targets, "priority", 7)
	if got[0].ModelID != "a" || got[1].ModelID != "b" {
		t.Fatalf("priority order changed: %+v", got)
	}
}
