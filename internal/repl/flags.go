package repl

import (
	"fmt"
	"strings"
)

// FlagSet is a tiny POSIX-style flag parser tailored to the CLI's
// limited needs. We deliberately avoid pflag/cobra to stay
// dependency-light. Supported forms:
//
//	--key=value   --key value   --flag (boolean)
//
// Unknown flags surface as errors. Non-flag tokens are returned in the
// Positional slice in order.
type FlagSet struct {
	String map[string]*string // pointer values populated on parse
	Bool   map[string]*bool

	Positional []string
}

// NewFlagSet seeds a FlagSet ready to be configured with the
// Register* methods.
func NewFlagSet() *FlagSet {
	return &FlagSet{
		String: map[string]*string{},
		Bool:   map[string]*bool{},
	}
}

// RegisterString declares a --name string flag. The returned pointer
// is populated after Parse.
func (f *FlagSet) RegisterString(name string) *string {
	v := new(string)
	f.String[name] = v
	return v
}

// RegisterBool declares a --name bool flag.
func (f *FlagSet) RegisterBool(name string) *bool {
	v := new(bool)
	f.Bool[name] = v
	return v
}

// Parse walks args, populating registered flags and collecting non-flag
// tokens into Positional.
func (f *FlagSet) Parse(args []string) error {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			f.Positional = append(f.Positional, a)
			continue
		}
		name := a[2:]
		var value string
		hasValue := false
		if idx := strings.IndexByte(name, '='); idx >= 0 {
			value = name[idx+1:]
			name = name[:idx]
			hasValue = true
		}
		if p, ok := f.Bool[name]; ok {
			if hasValue {
				switch strings.ToLower(value) {
				case "true", "1", "yes", "y":
					*p = true
				case "false", "0", "no", "n":
					*p = false
				default:
					return fmt.Errorf("flag --%s: expected boolean, got %q", name, value)
				}
				continue
			}
			*p = true
			continue
		}
		if p, ok := f.String[name]; ok {
			if hasValue {
				*p = value
				continue
			}
			if i+1 >= len(args) {
				return fmt.Errorf("flag --%s requires a value", name)
			}
			i++
			*p = args[i] //nolint:gosec // bounds checked one line above
			continue
		}
		return fmt.Errorf("unknown flag --%s", name)
	}
	return nil
}
