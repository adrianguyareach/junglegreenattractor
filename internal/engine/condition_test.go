package engine

import "testing"

func TestEvaluateConditionSuccess(t *testing.T) {
	outcome := &Outcome{Status: StatusSuccess}
	ctx := NewContext()

	if !EvaluateCondition("outcome=success", outcome, ctx) {
		t.Error("expected outcome=success to be true")
	}
}

func TestEvaluateConditionFail(t *testing.T) {
	outcome := &Outcome{Status: StatusFail}
	ctx := NewContext()

	if !EvaluateCondition("outcome=fail", outcome, ctx) {
		t.Error("expected outcome=fail to be true")
	}
	if EvaluateCondition("outcome=success", outcome, ctx) {
		t.Error("expected outcome=success to be false when status is fail")
	}
}

func TestEvaluateConditionNotEquals(t *testing.T) {
	outcome := &Outcome{Status: StatusSuccess}
	ctx := NewContext()

	if !EvaluateCondition("outcome!=fail", outcome, ctx) {
		t.Error("expected outcome!=fail to be true when status is success")
	}
	if EvaluateCondition("outcome!=success", outcome, ctx) {
		t.Error("expected outcome!=success to be false when status is success")
	}
}

func TestEvaluateConditionAND(t *testing.T) {
	outcome := &Outcome{Status: StatusSuccess}
	ctx := NewContext()
	ctx.Set("tests_passed", "true")

	if !EvaluateCondition("outcome=success && tests_passed=true", outcome, ctx) {
		t.Error("expected compound condition to be true")
	}

	ctx.Set("tests_passed", "false")
	if EvaluateCondition("outcome=success && tests_passed=true", outcome, ctx) {
		t.Error("expected compound condition to be false when one clause fails")
	}
}

func TestEvaluateConditionContextKey(t *testing.T) {
	outcome := &Outcome{Status: StatusSuccess}
	ctx := NewContext()
	ctx.Set("loop_state", "active")

	if !EvaluateCondition("context.loop_state=active", outcome, ctx) {
		t.Error("expected context.loop_state=active to be true")
	}
	if !EvaluateCondition("context.loop_state!=exhausted", outcome, ctx) {
		t.Error("expected context.loop_state!=exhausted to be true")
	}
}

func TestEvaluateConditionEmpty(t *testing.T) {
	outcome := &Outcome{Status: StatusFail}
	ctx := NewContext()

	if !EvaluateCondition("", outcome, ctx) {
		t.Error("empty condition should always be true")
	}
}

func TestEvaluateConditionPreferredLabel(t *testing.T) {
	outcome := &Outcome{Status: StatusSuccess, PreferredLabel: "Fix"}
	ctx := NewContext()

	if !EvaluateCondition("preferred_label=Fix", outcome, ctx) {
		t.Error("expected preferred_label=Fix to be true")
	}
}
