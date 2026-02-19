package auth

import (
	"testing"

	"document-mdp/ent/schema"
)

func TestEvaluateEffects_Precedence(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		deny, r, w := evaluateEffects(nil)
		if deny || r || w {
			t.Fatalf("expected all false, got deny=%v read=%v write=%v", deny, r, w)
		}
	})

	t.Run("read", func(t *testing.T) {
		deny, r, w := evaluateEffects([]schema.ACLEffect{schema.ACLEffectRead})
		if deny || !r || w {
			t.Fatalf("expected read only, got deny=%v read=%v write=%v", deny, r, w)
		}
	})

	t.Run("read_write", func(t *testing.T) {
		deny, r, w := evaluateEffects([]schema.ACLEffect{schema.ACLEffectWrite})
		if deny || !r || !w {
			t.Fatalf("expected read+write, got deny=%v read=%v write=%v", deny, r, w)
		}
	})

	t.Run("deny_overrides", func(t *testing.T) {
		deny, r, w := evaluateEffects([]schema.ACLEffect{schema.ACLEffectWrite, schema.ACLEffectRead, schema.ACLEffectDeny})
		if !deny || r || w {
			t.Fatalf("expected deny only, got deny=%v read=%v write=%v", deny, r, w)
		}
	})
}
