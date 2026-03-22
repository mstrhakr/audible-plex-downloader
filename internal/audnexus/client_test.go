package audnexus

import (
	"encoding/json"
	"testing"
)

func TestFlexibleFloat64UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    FlexibleFloat64
		wantErr bool
	}{
		{name: "json number", input: `4.7`, want: FlexibleFloat64(4.7)},
		{name: "quoted number", input: `"4.7"`, want: FlexibleFloat64(4.7)},
		{name: "quoted number with spaces", input: `" 4.7 "`, want: FlexibleFloat64(4.7)},
		{name: "null", input: `null`, want: FlexibleFloat64(0)},
		{name: "empty string", input: `""`, want: FlexibleFloat64(0)},
		{name: "invalid string", input: `"not-a-number"`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got FlexibleFloat64
			err := json.Unmarshal([]byte(tc.input), &got)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBookResponseRatingString(t *testing.T) {
	var got BookResponse
	payload := `{"asin":"B0FV3R3FM7","title":"Example","rating":"4.8"}`

	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Rating != FlexibleFloat64(4.8) {
		t.Fatalf("got rating %v, want %v", got.Rating, FlexibleFloat64(4.8))
	}
}

