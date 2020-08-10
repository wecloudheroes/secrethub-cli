package cli

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"bitbucket.org/zombiezen/cardcpx/natsort"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/xhit/go-str2duration"
)

var (
	// DefaultEnvSeparator defines how to join env var names.
	DefaultEnvSeparator = "_"
	// DefaultCommandDelimiters defines which delimiters should be replaced.
	DefaultCommandDelimiters = []string{" ", "-"}
)

// App represents a command-line application that wraps the
// kingpin library and adds additional functionality.
type App struct {
	Application      cobra.Command
	name             string
	delimiters       []string
	separator        string
	knownEnvVars     map[string]struct{}
	extraEnvVarFuncs []func(key string) bool
}

// NewApp defines a new command-line application.
func NewApp(name, help string) *App {
	return &App{
		Application:      cobra.Command{Use: name, Short: help},
		name:             formatName(name, "", DefaultEnvSeparator, DefaultCommandDelimiters...),
		delimiters:       DefaultCommandDelimiters,
		separator:        DefaultEnvSeparator,
		knownEnvVars:     make(map[string]struct{}),
		extraEnvVarFuncs: []func(string) bool{},
	}
}

// Command defines a new top-level command with the given name and help text.
func (a *App) CreateCommand(name, help string) *CommandClause {
	return &CommandClause{
		Command: func() *cobra.Command {
			newCommand := &cobra.Command{Use: name, Short: help}
			a.Application.AddCommand(newCommand)
			return newCommand
		}(),
		name: name,
		app:  a,
	}
}

//
//// Version adds a flag for displaying the application version number.
//func (a *App) Version(version string) *App {
//	a.Application = a.Application.Version(version)
//	return a
//}
//
// Flag defines a new flag with the given long name and help text,
// adding an environment variable default configurable by APP_FLAG_NAME.
func (a *App) Flag(name, help string) *Flag {
	envVar := formatName(name, a.name, a.separator, a.delimiters...)
	a.registerEnvVar(envVar)
	flag := Flag{
		Flag:   &pflag.Flag{Name: name, Usage: help},
		envVar: envVar,
		app:    a,
	}
	return flag.Envar(envVar)
}

// registerEnvVar ensures the App recognizes an environment variable.
func (a *App) registerEnvVar(name string) {
	a.knownEnvVars[strings.ToUpper(name)] = struct{}{}
}

// unregisterEnvVar ensures the App does not recognize an environment variable.
func (a *App) unregisterEnvVar(name string) {
	delete(a.knownEnvVars, strings.ToUpper(name))
}

// ExtraEnvVarFunc takes a function that determines additional environment variables
// recognized by the application.
func (a *App) ExtraEnvVarFunc(f func(key string) bool) *App {
	if f != nil {
		a.extraEnvVarFuncs = append(a.extraEnvVarFuncs, f)
	}
	return a
}

func (a *App) isExtraEnvVar(key string) bool {
	for _, check := range a.extraEnvVarFuncs {
		if check(key) {
			return true
		}
	}
	return false
}

// PrintEnv reads all environment variables starting with the app name and writes
// a table with the keys and their status: set, empty, unrecognized. The value
// of environment variables are not printed out for security reasons. The list
// is limited to variables that are actually set in the environment. Setting
// verbose to true will also include all known variables that are not set.
func (a *App) PrintEnv(w io.Writer, verbose bool, osEnv func() []string) error {
	tabWriter := tabwriter.NewWriter(w, 0, 4, 4, ' ', 0)
	fmt.Fprintf(tabWriter, "%s\t%s\n", "NAME", "STATUS")

	envVarStatus := make(map[string]string)
	for _, envVar := range osEnv() {
		key, _, match := splitVar(a.name, a.separator, envVar)
		key = strings.ToUpper(key)
		if match {
			_, isKnown := a.knownEnvVars[key]
			if isKnown || a.isExtraEnvVar(key) {
				envVarStatus[key] = "set"
			} else {
				envVarStatus[key] = "unrecognized"
			}
		}
	}

	if verbose {
		for known := range a.knownEnvVars {
			_, isSet := envVarStatus[known]
			if !isSet {
				envVarStatus[known] = "-"
			}
		}
	}

	rows := []string{}
	for envVar, status := range envVarStatus {
		rows = append(rows, fmt.Sprintf("%s\t%s", envVar, status))
	}

	natsort.Strings(rows)
	for _, row := range rows {
		fmt.Fprintln(tabWriter, row)
	}

	err := tabWriter.Flush()
	if err != nil {
		return err
	}

	return nil
}

// CheckStrictEnv checks that every environment variable that starts with the app name is recognized by the application.
func (a *App) CheckStrictEnv() error {
	for _, envVar := range os.Environ() {
		key, _, match := splitVar(a.name, a.separator, envVar)
		if match {
			key = strings.ToUpper(key)
			_, isKnown := a.knownEnvVars[key]
			if !(isKnown || a.isExtraEnvVar(key)) {
				return fmt.Errorf("environment variable set, but not recognized: %s", key)
			}
		}
	}
	return nil
}

// CommandClause represents a command clause in a command0-line application.
type CommandClause struct {
	*cobra.Command

	name string
	app  *App
}

// Command adds a new subcommand to this command.
func (cmd *CommandClause) CreateCommand(name, help string) *CommandClause {
	return &CommandClause{
		Command: func() *cobra.Command {
			newCommand := &cobra.Command{Use: name, Short: help}
			return newCommand
		}(),
		name: name,
		app:  cmd.app,
	}
}

// Hidden hides the command in help texts.
func (cmd *CommandClause) Hidden() *CommandClause {
	cmd.Command.Hidden = true
	return cmd
}

func (cmd *CommandClause) FullCommand() string {
	return strings.Join(os.Args[:], " ")
}

func (cmd *CommandClause) HelpLong(helpLong string) {
	cmd.Long = helpLong
}

func (cmd *CommandClause) Alias(alias string) {
	if cmd.Aliases == nil {
		cmd.Aliases = []string{alias}
	} else {
		cmd.Aliases = append(cmd.Aliases, alias)
	}
}

// Flag defines a new flag with the given long name and help text,
// adding an environment variable default configurable by APP_COMMAND_FLAG_NAME.
// The help text is suffixed with a description of secrthe environment variable default.
func (cmd *CommandClause) Flag(name, help string) *Flag {
	fullCmd := strings.Replace(cmd.FullCommand(), " ", cmd.app.separator, -1)
	prefix := formatName(fullCmd, cmd.app.name, cmd.app.separator, cmd.app.delimiters...)
	envVar := formatName(name, prefix, cmd.app.separator, cmd.app.delimiters...)

	cmd.app.registerEnvVar(envVar)
	flag := &pflag.Flag{Name: name, Usage: help}
	return &Flag{
		Flag:   flag,
		app:    cmd.app,
		envVar: envVar,
	}
}

// Flag represents a command-line flag.
type Flag struct {
	*pflag.Flag

	envVar string
	app    *App
}

// Envar overrides the environment variable name that configures the default
// value for a flag.
func (f *Flag) Envar(name string) *Flag {
	name = strings.ToUpper(name)
	if f.envVar != "" {
		f.app.unregisterEnvVar(f.envVar)
	}
	f.app.registerEnvVar(name)
	f.envVar = name
	f.Flag.DefValue = os.Getenv(f.envVar)
	return f
}

// NoEnvar forces environment variable defaults to be disabled for this flag.
func (f *Flag) NoEnvar() *Flag {
	if f.envVar != "" {
		f.app.unregisterEnvVar(f.envVar)
	}
	f.envVar = ""
	f.Flag.DefValue = ""
	return f
}

// Hidden hides the flag in help texts.
func (f *Flag) Hidden() *Flag {
	f.Flag.Hidden = true
	return f
}

// Short puts the shorthand of the flag.
func (f *Flag) Short(s rune) *Flag {
	f.Flag.Shorthand = string(s)
	return f
}

func (f *Flag) Default(val string) *Flag {
	if f.Flag.DefValue == "" {
		f.Flag.DefValue = val
	}
	return f
}

func (f *Flag) PlaceHolder(val string) *Flag {
	return f
}

// TODO Implement it properly
func (f *Flag) HintOptions(options ...string) *Flag {
	return f
}

// TODO Implement it properly
func (f *Flag) SetValue(location interface{}) *Flag {
	if f.Value != nil {
		location = &f.Value
	}
	return f
}

// Hidden hides the command in help texts.
func (f *Flag) BoolVar(location *bool) *Flag {
	if f.Value != nil {
		intermediary, _ := strconv.ParseBool(f.Value.String())
		*location = intermediary
	} else {
		*location = false
	}

	return f
}

func (f *Flag) StringVar(location *string) *Flag {
	if f.Value != nil {
		*location = f.Value.String()
	} else {
		*location = f.DefValue
	}
	return f
}

// TODO Implement the following functions properly
func (f *Flag) DurationVar(location *time.Duration) *Flag {
	if f.Value != nil {
		*location, _ = str2duration.Str2Duration(f.Value.String())
	} else {
		*location, _ = str2duration.Str2Duration(f.DefValue)
	}
	return f
}

func (f *Flag) URLVar(location **url.URL) *Flag {
	if f.Value != nil {
		parsedURL, _ := url.Parse(f.Value.String())
		*location = parsedURL
	} else {
		parsedURL, _ := url.Parse(f.DefValue)
		location = &parsedURL
	}
	return f
}

func (f *Flag) IntVar(location *int) *Flag {
	if f.Value != nil {
		*location, _ = strconv.Atoi(f.Value.String())
	} else {
		*location, _ = strconv.Atoi(f.DefValue)
	}
	return f
}

func (f *Flag) ExistingFileVar(location *string) *Flag {
	return f
}

func (f *Flag) StringMapVar(location *map[string]string) *Flag {
	return f
}

// End of TODO

// formatName takes a name and converts it to an uppercased name,
// joined by the given separator and prefixed with the given prefix.
func formatName(name, prefix, separator string, delimiters ...string) string {
	for _, delim := range delimiters {
		name = strings.Replace(name, delim, separator, -1)
	}

	if prefix == "" {
		return strings.ToUpper(name)
	}

	return strings.ToUpper(strings.Join([]string{prefix, name}, separator))
}

// splitVar gets the key value pair from a key=value declaration, returning
// true if it matches the given prefix.
func splitVar(prefix, separator, envVar string) (string, string, bool) {
	envVar = strings.TrimSpace(envVar)
	split := strings.Split(envVar, "=")
	if len(split) != 2 {
		return "", "", false
	}

	prefix = fmt.Sprintf("%s%s", strings.ToUpper(prefix), separator)
	return split[0], split[1], strings.HasPrefix(strings.ToUpper(split[0]), prefix)
}
