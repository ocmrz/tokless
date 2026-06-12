package util

import (
	"testing"
)

func TestInstallSucceeded(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		actual     *string
		wantResult string
		wantOk     bool
	}{
		{
			name:       "nil actual",
			target:     "1.2.3",
			actual:     nil,
			wantResult: "",
			wantOk:     false,
		},
		{
			name:       "exact match",
			target:     "1.2.3",
			actual:     stringPtr("1.2.3"),
			wantResult: "1.2.3",
			wantOk:     true,
		},
		{
			name:       "target empty",
			target:     "",
			actual:     stringPtr("1.2.3"),
			wantResult: "1.2.3",
			wantOk:     true,
		},
		{
			name:       "mismatch",
			target:     "2.0.0",
			actual:     stringPtr("1.0.0"),
			wantResult: "",
			wantOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotOk := installSucceeded(tt.target, tt.actual)
			if gotResult != tt.wantResult {
				t.Errorf("installSucceeded() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if gotOk != tt.wantOk {
				t.Errorf("installSucceeded() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
