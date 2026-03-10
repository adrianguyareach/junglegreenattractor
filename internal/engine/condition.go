package engine

import "strings"

// EvaluateCondition evaluates a condition expression against the outcome and context.
// Grammar: clause ( '&&' clause )*
// Clause:  key '=' value | key '!=' value
func EvaluateCondition(condition string, outcome *Outcome, ctx *Context) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true
	}

	clauses := strings.Split(condition, "&&")
	for _, clause := range clauses {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}
		if !evaluateClause(clause, outcome, ctx) {
			return false
		}
	}
	return true
}

func evaluateClause(clause string, outcome *Outcome, ctx *Context) bool {
	if idx := strings.Index(clause, "!="); idx >= 0 {
		key := strings.TrimSpace(clause[:idx])
		val := strings.TrimSpace(clause[idx+2:])
		return resolveKey(key, outcome, ctx) != val
	}
	if idx := strings.Index(clause, "="); idx >= 0 {
		key := strings.TrimSpace(clause[:idx])
		val := strings.TrimSpace(clause[idx+1:])
		return resolveKey(key, outcome, ctx) == val
	}
	// Bare key: truthy check
	return resolveKey(strings.TrimSpace(clause), outcome, ctx) != ""
}

func resolveKey(key string, outcome *Outcome, ctx *Context) string {
	switch key {
	case "outcome":
		if outcome != nil {
			return string(outcome.Status)
		}
		return ""
	case "preferred_label":
		if outcome != nil {
			return outcome.PreferredLabel
		}
		return ""
	}

	if strings.HasPrefix(key, "context.") {
		ctxKey := key
		if v := ctx.Get(ctxKey); v != "" {
			return v
		}
		stripped := strings.TrimPrefix(key, "context.")
		return ctx.Get(stripped)
	}

	return ctx.Get(key)
}
