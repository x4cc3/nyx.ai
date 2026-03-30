package queue

import "testing"

func TestSubjectToken(t *testing.T) {
	if got := subjectToken("flow.alpha beta"); got != "flow_alpha_beta" {
		t.Fatalf("unexpected token: %s", got)
	}
	if got := subjectToken("   "); got != "unknown" {
		t.Fatalf("expected unknown token, got %s", got)
	}
}

func TestDerivedSubjects(t *testing.T) {
	transport := &JetStreamTransport{
		actionResultSubject: "nyx.actions.result",
		eventSubject:        "nyx.events.flow",
		dlqSubject:          "nyx.dlq",
	}

	if got := transport.actionResultSubjectFor("action.123"); got != "nyx.actions.result.action_123" {
		t.Fatalf("unexpected action result subject: %s", got)
	}
	if got := transport.eventSubjectFor("flow.123"); got != "nyx.events.flow.flow_123" {
		t.Fatalf("unexpected event subject: %s", got)
	}
	if got := transport.deadLetterSubjectFor("action.request"); got != "nyx.dlq.action_request" {
		t.Fatalf("unexpected dlq subject: %s", got)
	}
}
