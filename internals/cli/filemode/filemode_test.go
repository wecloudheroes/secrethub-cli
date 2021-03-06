package filemode_test

import (
	"os"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/cli/filemode"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func TestParseFilemode(t *testing.T) {
	cases := map[string]struct {
		input    string
		expected os.FileMode
		error    error
	}{
		"empty": {
			input:    "",
			expected: 0,
			error:    filemode.ErrInvalidFilemode("", "filemodes must contain at least three digits"),
		},
		"missing_trailing_zero": {
			input:    "44",
			expected: 0,
			error:    filemode.ErrInvalidFilemode("44", "filemodes must contain at least three digits"),
		},
		"valid": {
			input:    "0660",
			expected: 0660,
			error:    nil,
		},
		"double": {
			input:    "0440",
			expected: 0440,
			error:    nil,
		},
		"missing_leading_zero": {
			input:    "440",
			expected: 0440,
			error:    nil,
		},
		// TODO SHDEV-1029: Add case where ParseUint returns an error.
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			actual, err := filemode.Parse(tc.input)

			// Assert
			assert.Equal(t, err, tc.error)
			assert.Equal(t, actual, tc.expected)
		})
	}
}
