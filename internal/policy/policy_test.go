package policy

import "testing"

func TestEvaluateUsesMostSpecificRule(t *testing.T) {
	rules := []Rule{
		{PathPrefix: "/Users/me/Prog", State: NeedsReview},
		{PathPrefix: "/Users/me/Prog/FH/public-tool", State: AutoApproved, DisplayName: "Public Tool"},
	}

	decision := Evaluate("/Users/me/Prog/FH/public-tool/subdir", rules)
	if decision.State != AutoApproved {
		t.Fatalf("state = %s, want %s", decision.State, AutoApproved)
	}
	if decision.DisplayName != "Public Tool" {
		t.Fatalf("display name = %q", decision.DisplayName)
	}
}

func TestEvaluateDefaultsUnknownToReview(t *testing.T) {
	decision := Evaluate("/Users/me/Prog/unknown", nil)
	if decision.State != NeedsReview {
		t.Fatalf("state = %s, want %s", decision.State, NeedsReview)
	}
}
