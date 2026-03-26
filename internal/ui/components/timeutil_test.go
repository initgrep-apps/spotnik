package components_test

import (
	"testing"
	"time"

	"github.com/initgrep-apps/spotnik/internal/ui/components"
	"github.com/stretchr/testify/assert"
)

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		input time.Time
		want  string
	}{
		{
			name:  "just now",
			input: now.Add(-30 * time.Second),
			want:  "just now",
		},
		{
			name:  "5 minutes ago",
			input: now.Add(-5 * time.Minute),
			want:  "5 min ago",
		},
		{
			name:  "1 minute ago",
			input: now.Add(-61 * time.Second),
			want:  "1 min ago",
		},
		{
			name:  "2 hours ago",
			input: now.Add(-2 * time.Hour),
			want:  "2 hr ago",
		},
		{
			name:  "23 hours ago",
			input: now.Add(-23 * time.Hour),
			want:  "23 hr ago",
		},
		{
			name:  "3 days ago",
			input: now.Add(-3 * 24 * time.Hour),
			want:  "3 days ago",
		},
		{
			name:  "6 days ago",
			input: now.Add(-6 * 24 * time.Hour),
			want:  "6 days ago",
		},
		{
			name:  "7+ days returns short date",
			input: now.Add(-7 * 24 * time.Hour),
			// We don't assert the exact date (it's day-dependent),
			// just that it does not contain "ago".
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := components.FormatRelativeTime(tt.input)
			if tt.name == "7+ days returns short date" {
				assert.NotContains(t, got, "ago")
				assert.NotEmpty(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
