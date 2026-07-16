package branding

import "testing"

func TestResolveProductName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", want: DefaultProductName},
		{name: "blank", input: "  ", want: DefaultProductName},
		{name: "upstream default", input: UpstreamDefaultProductName, want: DefaultProductName},
		{name: "custom", input: "  My Console  ", want: "My Console"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveProductName(tt.input); got != tt.want {
				t.Fatalf("ResolveProductName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
