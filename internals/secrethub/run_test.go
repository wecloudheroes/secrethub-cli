package secrethub

import (
	"errors"
	"testing"

	generictpl "github.com/secrethub/secrethub-cli/internals/tpl"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func elemEqual(t *testing.T, actual []envvar, expected []envvar) {
isExpected:
	for _, a := range actual {
		for _, e := range expected {
			if a == e {
				continue isExpected
			}
		}
		t.Errorf("%+v encountered but not expected", a)
	}

isEncountered:
	for _, e := range expected {
		for _, a := range actual {
			if a == e {
				continue isEncountered
			}
		}
		t.Errorf("%+v expected but not encountered", e)
	}
}

func TestParseEnv(t *testing.T) {
	cases := map[string]struct {
		raw      string
		expected []envvar
		err      error
	}{
		"success": {
			raw: "foo=bar\nbaz={{path/to/secret}}",
			expected: []envvar{
				{
					key:        "foo",
					value:      "bar",
					lineNumber: 1,
				},
				{
					key:        "baz",
					value:      "{{path/to/secret}}",
					lineNumber: 2,
				},
			},
		},
		"success with spaces": {
			raw: "key = value",
			expected: []envvar{
				{
					key:        "key",
					value:      "value",
					lineNumber: 1,
				},
			},
		},
		"success with multiple spaces": {
			raw: "key    = value",
			expected: []envvar{
				{
					key:        "key",
					value:      "value",
					lineNumber: 1,
				},
			},
		},
		"success with tabs": {
			raw: "key\t=\tvalue",
			expected: []envvar{
				{
					key:        "key",
					value:      "value",
					lineNumber: 1,
				},
			},
		},
		"success with single quotes": {
			raw: "key='value'",
			expected: []envvar{
				{
					key:        "key",
					value:      "value",
					lineNumber: 1,
				},
			},
		},
		"success with double quotes": {
			raw: `key="value"`,
			expected: []envvar{
				{
					key:        "key",
					value:      "value",
					lineNumber: 1,
				},
			},
		},
		"success with quotes and whitespace": {
			raw: "key = 'value'",
			expected: []envvar{
				{
					key:        "key",
					value:      "value",
					lineNumber: 1,
				},
			},
		},
		"success comment": {
			raw: "# database\nDB_USER = user\nDB_PASS = pass",
			expected: []envvar{
				{
					key:        "DB_USER",
					value:      "user",
					lineNumber: 2,
				},
				{
					key:        "DB_PASS",
					value:      "pass",
					lineNumber: 3,
				},
			},
		},
		"success comment prefixed with spaces": {
			raw: "    # database\nDB_USER = user\nDB_PASS = pass",
			expected: []envvar{
				{
					key:        "DB_USER",
					value:      "user",
					lineNumber: 2,
				},
				{
					key:        "DB_PASS",
					value:      "pass",
					lineNumber: 3,
				},
			},
		},
		"success comment prefixed with a tab": {
			raw: "\t# database\nDB_USER = user\nDB_PASS = pass",
			expected: []envvar{
				{
					key:        "DB_USER",
					value:      "user",
					lineNumber: 2,
				},
				{
					key:        "DB_PASS",
					value:      "pass",
					lineNumber: 3,
				},
			},
		},
		"success empty lines": {
			raw: "foo=bar\n\nbar=baz",
			expected: []envvar{
				{
					key:        "foo",
					value:      "bar",
					lineNumber: 1,
				},
				{
					key:        "bar",
					value:      "baz",
					lineNumber: 3,
				},
			},
		},
		"success line with only spaces": {
			raw: "foo=bar\n    \nbar = baz",
			expected: []envvar{
				{
					key:        "foo",
					value:      "bar",
					lineNumber: 1,
				},
				{
					key:        "bar",
					value:      "baz",
					lineNumber: 3,
				},
			},
		},
		"= sign in value": {
			raw: "foo=foo=bar",
			expected: []envvar{
				{
					key:        "foo",
					value:      "foo=bar",
					lineNumber: 1,
				},
			},
		},
		"invalid": {
			raw: "foobar",
			err: ErrTemplate(1, errors.New("template is not formatted as key=value pairs")),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual, err := parseEnv(tc.raw)

			elemEqual(t, actual, tc.expected)
			assert.Equal(t, err, tc.err)
		})
	}
}

func TestParseYML(t *testing.T) {
	cases := map[string]struct {
		raw      string
		expected []envvar
		err      error
	}{
		"success": {
			raw: "foo: bar\nbaz: ${path/to/secret}",
			expected: []envvar{
				{
					key:        "foo",
					value:      "bar",
					lineNumber: -1,
				},
				{
					key:        "baz",
					value:      "${path/to/secret}",
					lineNumber: -1,
				},
			},
		},
		"= in value": {
			raw: "foo: foo=bar\nbar: baz",
			expected: []envvar{
				{
					key:        "foo",
					value:      "foo=bar",
					lineNumber: -1,
				},
				{
					key:        "bar",
					value:      "baz",
					lineNumber: -1,
				},
			},
		},
		"nested yml": {
			raw: "ROOT:\n\tSUB\n\t\tNAME: val1",
			err: errors.New("yaml: line 2: found character that cannot start any token"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual, err := parseYML(tc.raw)

			elemEqual(t, actual, tc.expected)
			assert.Equal(t, err, tc.err)
		})
	}
}

func TestNewEnv(t *testing.T) {
	cases := map[string]struct {
		raw          string
		replacements map[string]string
		templateVars map[string]string
		expected     map[string]string
		err          error
	}{
		"success": {
			raw: "foo=bar\nbaz={{path/to/secret}}",
			replacements: map[string]string{
				"path/to/secret": "val",
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "val",
			},
		},
		"success with vars": {
			raw: "foo=bar\nbaz={{${app}/db/pass}}",
			replacements: map[string]string{
				"company/application/db/pass": "secret",
			},
			templateVars: map[string]string{
				"app": "company/application",
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "secret",
			},
		},
		"success yml": {
			raw: "foo: bar\nbaz: ${path/to/secret}",
			replacements: map[string]string{
				"path/to/secret": "val",
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "val",
			},
		},
		"yml template error": {
			raw: "foo: bar: baz",
			err: ErrTemplate(1, errors.New("template is not formatted as key=value pairs")),
		},
		"yml secret template error": {
			raw: "foo: ${path/to/secret",
			err: generictpl.ErrTagNotClosed("}"),
		},
		"env error": {
			raw: "foo={{path/to/secret",
			err: ErrTemplate(1, generictpl.ErrTagNotClosed("}}")),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			env, err := NewEnv(tc.raw, tc.templateVars)
			if err != nil {
				assert.Equal(t, err, tc.err)
			} else {
				actual, err := env.Env(tc.replacements)
				assert.Equal(t, err, tc.err)

				assert.Equal(t, actual, tc.expected)
			}
		})
	}
}

func TestRunCommand_Run(t *testing.T) {
	cases := map[string]struct {
		command RunCommand
		err     error
	}{
		"invalid template var: start with a number": {
			command: RunCommand{
				templateVars: map[string]string{
					"0foo": "value",
				},
				envar: map[string]string{},
			},
			err: ErrInvalidTemplateVar("0foo"),
		},
		"invalid template var: illegal character": {
			command: RunCommand{
				templateVars: map[string]string{
					"foo@bar": "value",
				},
				envar: map[string]string{},
			},
			err: ErrInvalidTemplateVar("foo@bar"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.command.Run()
			assert.Equal(t, err, tc.err)
		})
	}
}

func TestTrimQuotes(t *testing.T) {
	cases := map[string]struct {
		in       string
		expected string
	}{
		"unquoted": {
			in:       `foo`,
			expected: `foo`,
		},
		"single quoted": {
			in:       `'foo'`,
			expected: `foo`,
		},
		"double quoted": {
			in:       `"foo"`,
			expected: `foo`,
		},
		"empty string": {
			in:       "",
			expected: "",
		},
		"single quoted empty string": {
			in:       `''`,
			expected: ``,
		},
		"double qouted empty string": {
			in:       `""`,
			expected: ``,
		},
		"single quote wrapped in single quote": {
			in:       `''foo''`,
			expected: `'foo'`,
		},
		"single quote wrapped in double quote": {
			in:       `"'foo'"`,
			expected: `'foo'`,
		},
		"double quote wrapped in double quote": {
			in:       `""foo""`,
			expected: `"foo"`,
		},
		"double quote wrapped in single quote": {
			in:       `'"foo"'`,
			expected: `"foo"`,
		},
		"single quote opened but not closed": {
			in:       `'foo`,
			expected: `'foo`,
		},
		"double quote opened but not closed": {
			in:       `"foo`,
			expected: `"foo`,
		},
		"single quote closed but not opened": {
			in:       `foo'`,
			expected: `foo'`,
		},
		"double quote closed but not opened": {
			in:       `foo"`,
			expected: `foo"`,
		},

		"single quoted with inner leading whitespace": {
			in:       `' foo'`,
			expected: ` foo`,
		},
		"double quoted with inner leading whitespace": {
			in:       `" foo"`,
			expected: ` foo`,
		},
		"single quoted with inner trailing whitespace": {
			in:       `'foo '`,
			expected: `foo `,
		},
		"double quoted with inner trailing whitespace": {
			in:       `"foo "`,
			expected: `foo `,
		},

		// Trimming OUTER whitespace is explicitly not the responsibility of this function.
		"single quoted with outer leading whitespace": {
			in:       ` 'foo'`,
			expected: ` 'foo'`,
		},
		"double quoted with outer leading whitespace": {
			in:       ` "foo"`,
			expected: ` "foo"`,
		},
		"single quoted with outer trailing whitespace": {
			in:       `'foo' `,
			expected: `'foo' `,
		},
		"double quoted with outer trailing whitespace": {
			in:       `"foo" `,
			expected: `"foo" `,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual := trimQuotes(tc.in)

			assert.Equal(t, actual, tc.expected)
		})
	}
}
