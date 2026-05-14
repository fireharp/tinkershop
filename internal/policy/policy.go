package policy

import (
	"path/filepath"
	"strings"
)

type State string

const (
	Blocked      State = "blocked"
	NeedsReview  State = "needs_review"
	AutoApproved State = "auto_approved"
)

type Rule struct {
	PathPrefix  string   `json:"path_prefix"`
	DisplayName string   `json:"display_name,omitempty"`
	State       State    `json:"state"`
	Redact      []string `json:"redact,omitempty"`
}

type Decision struct {
	State       State
	DisplayName string
	Rule        *Rule
}

func Evaluate(path string, rules []Rule) Decision {
	cleanPath := filepath.Clean(path)
	var best *Rule
	bestLen := -1

	for i := range rules {
		rule := &rules[i]
		if rule.PathPrefix == "" {
			continue
		}
		prefix := filepath.Clean(rule.PathPrefix)
		if cleanPath == prefix || strings.HasPrefix(cleanPath, prefix+string(filepath.Separator)) {
			if len(prefix) > bestLen {
				best = rule
				bestLen = len(prefix)
			}
		}
	}

	if best == nil {
		return Decision{State: NeedsReview}
	}

	state := best.State
	if state == "" {
		state = NeedsReview
	}

	return Decision{
		State:       state,
		DisplayName: best.DisplayName,
		Rule:        best,
	}
}
