package util

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		// Standard durations (time.ParseDuration)
		{"seconds", "5s", 5 * time.Second, false},
		{"hour", "1h", time.Hour, false},
		{"hour and minutes", "1h30m", 90 * time.Minute, false},
		{"168 hours", "168h", 168 * time.Hour, false},
		{"minutes and seconds", "2m30s", 150 * time.Second, false},
		{"milliseconds", "500ms", 500 * time.Millisecond, false},

		// Day suffix
		{"1 day", "1d", 24 * time.Hour, false},
		{"7 days", "7d", 168 * time.Hour, false},
		{"1 day 12 hours", "1d12h", 36 * time.Hour, false},
		{"1 day 12 hours 30 minutes", "1d12h30m", 36*time.Hour + 30*time.Minute, false},
		{"3 days 5 minutes", "3d5m", 3*24*time.Hour + 5*time.Minute, false},

		// Zero and empty
		{"empty string", "", 0, false},
		{"zero seconds", "0s", 0, false},
		{"zero days", "0d", 0, false},

		// Invalid inputs
		{"non-numeric", "abc", 0, true},
		{"no unit", "7", 0, true},
		{"unknown unit", "1x", 0, true},
		{"negative days", "-1d", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMustParseDuration(t *testing.T) {
	t.Run("valid duration", func(t *testing.T) {
		d := MustParseDuration("1d")
		if d != 24*time.Hour {
			t.Errorf("MustParseDuration(\"1d\") = %v, want %v", d, 24*time.Hour)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		d := MustParseDuration("")
		if d != 0 {
			t.Errorf("MustParseDuration(\"\") = %v, want 0", d)
		}
	})

	t.Run("invalid input panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustParseDuration(\"abc\") expected panic")
			}
		}()
		MustParseDuration("abc")
	})
}
