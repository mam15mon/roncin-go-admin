package data

import (
	"testing"
	"unicode/utf8"
)

func TestClampNotificationBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxBytes int
		want     string
	}{
		{
			name:     "empty string",
			input:    "",
			maxBytes: 64,
			want:     "",
		},
		{
			name:     "zero maxBytes",
			input:    "hello",
			maxBytes: 0,
			want:     "",
		},
		{
			name:     "negative maxBytes",
			input:    "hello",
			maxBytes: -1,
			want:     "",
		},
		{
			name:     "ascii within limit",
			input:    "hello world",
			maxBytes: 20,
			want:     "hello world",
		},
		{
			name:     "ascii exact limit",
			input:    "12345",
			maxBytes: 5,
			want:     "12345",
		},
		{
			name:     "ascii exceeds limit",
			input:    "hello world",
			maxBytes: 5,
			want:     "hello",
		},
		{
			name:     "chinese within limit",
			input:    "成都分公司",
			maxBytes: 64,
			want:     "成都分公司",
		},
		{
			name:     "chinese exact limit",
			input:    "成都分公司", // 5 * 3 = 15 bytes
			maxBytes: 15,
			want:     "成都分公司",
		},
		{
			name:     "chinese truncated on rune boundary",
			input:    "成都分公司", // 15 bytes
			maxBytes: 14,      // 14 is inside the 5th character ("司", bytes 12..14)
			want:     "成都分公",  // 4 * 3 = 12 bytes
		},
		{
			name:     "chinese truncated on rune boundary 2",
			input:    "成都分公司",
			maxBytes: 13,
			want:     "成都分公",
		},
		{
			name:     "chinese truncated on rune boundary 3",
			input:    "成都分公司",
			maxBytes: 12,
			want:     "成都分公",
		},
		{
			name:     "chinese truncated on rune boundary 4",
			input:    "成都分公司",
			maxBytes: 11,
			want:     "成都分",
		},
		{
			name:     "chinese with whitespace trimmed",
			input:    "   成都分公司   ",
			maxBytes: 15,
			want:     "成都分公司",
		},
		{
			name:     "long escalated org suffix",
			input:    "成都分公司（该组织暂无管理员，由总部代管审批）",
			maxBytes: 256,
			want:     "成都分公司（该组织暂无管理员，由总部代管审批）",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := clampNotificationBytes(tc.input, tc.maxBytes)
			if got != tc.want {
				t.Fatalf("clampNotificationBytes(%q, %d) = %q, want %q", tc.input, tc.maxBytes, got, tc.want)
			}
			if len(got) > tc.maxBytes && tc.maxBytes > 0 {
				t.Fatalf("len(got)=%d > maxBytes=%d", len(got), tc.maxBytes)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("got %q is not valid UTF-8", got)
			}
		})
	}
}
