package auth

import "testing"

func TestExtractBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{name: "valid", header: "Bearer token123", want: "token123"},
		{name: "lowercase bearer", header: "bearer token123", want: "token123"},
		{name: "missing", header: "", wantErr: true},
		{name: "invalid scheme", header: "Basic abc", wantErr: true},
		{name: "missing token", header: "Bearer", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := extractBearerToken(tt.header)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBilletsFromContextEmpty(t *testing.T) {
	t.Parallel()
	if got := BilletsFromContext(t.Context()); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
