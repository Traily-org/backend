package ban

import "testing"

func TestNormalizeEmail_Variants_ReturnCanonicalForm(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{"lowercased and trimmed", "  Jane@Example.COM ", "jane@example.com"},
		{"plus tag removed", "jane+traily@example.com", "jane@example.com"},
		{"dots kept outside gmail", "jane.doe@example.com", "jane.doe@example.com"},
		{"gmail dots removed", "j.a.n.e@gmail.com", "jane@gmail.com"},
		{"gmail dots and tag", "Jane.Doe+spam@Gmail.com", "janedoe@gmail.com"},
		{"googlemail folded into gmail", "jane.doe@googlemail.com", "janedoe@gmail.com"},
		{"leading plus kept", "+jane@example.com", "+jane@example.com"},
		{"no at sign", "not-an-email", "not-an-email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeEmail(tt.email); got != tt.want {
				t.Errorf("NormalizeEmail(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}

func TestHashEmail_SameMailbox_SameHash(t *testing.T) {
	want := HashEmail("janedoe@gmail.com")

	for _, email := range []string{"Jane.Doe@gmail.com", "janedoe+1@googlemail.com", " j.a.n.e.d.o.e@GMAIL.com"} {
		if got := HashEmail(email); got != want {
			t.Errorf("HashEmail(%q) = %q, want %q", email, got, want)
		}
	}
	if HashEmail("other@gmail.com") == want {
		t.Error("HashEmail() of another mailbox must differ")
	}
	if len(want) != 64 {
		t.Errorf("HashEmail() length = %d, want 64 hex characters", len(want))
	}
}
