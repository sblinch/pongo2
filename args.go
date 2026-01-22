package pongo2

import (
	"errors"
	"fmt"
	"strconv"
)

// Args contains the arguments passed to filters
type Args struct {
	args   []*Value
	kwArgs map[string]*Value
}

// NewArgs creates a new Args object containing the specified arguments. named and args may both be nil
// (the zero value of Args is valid).
func NewArgs(named map[string]*Value, args ...*Value) *Args {
	return &Args{
		args:   args,
		kwArgs: named,
	}
}

// Len returns the number of positional arguments
func (a *Args) Len() int {
	return len(a.args)
}

// First returns the first positional argument (or an empty Value if no arguments exist)
func (a *Args) First() *Value {
	return a.Value(0)
}

// Get returns the positional argument at index i or (if index i does not exist) the named argument with the given name.
// If neither exists, an empty Value is returned.
func (a *Args) Get(i int, name string) *Value {
	v, _ := a.GetExists(i, name)
	return v
}

func (a *Args) GetDefault(i int, name string, dfl interface{}) *Value {
	v, exists := a.GetExists(i, name)
	if !exists {
		v = AsValue(dfl)
	}
	return v
}

// GetExists returns the positional argument at index i or (if index i does not exist) the named argument with the given
// name, and a boolean indicating whether the argument existed. If neither exists, an empty Value is returned.
func (a *Args) GetExists(i int, name string) (*Value, bool) {
	if i >= 0 && i < len(a.args) {
		return a.args[i], true
	}
	if len(a.kwArgs) > 0 {
		if v, exists := a.kwArgs[name]; exists {
			return v, true
		}
	}
	return emptyValue, false
}

// Value returns the positional argument at index i (or an empty Value if the index is invalid)
func (a *Args) Value(i int) *Value {
	v, _ := a.ValueExists(i)
	return v
}

var emptyValue = &Value{}

// ValueExists returns the positional argument at index i (or an empty Value if the index is invalid) and a boolean
// indicating whether the index was valid
func (a *Args) ValueExists(i int) (*Value, bool) {
	if a != nil && i >= 0 && i < len(a.args) {
		return a.args[i], true
	} else {
		return emptyValue, false
	}
}

// Values returns the raw list of positional arguments. This may be nil if no position arguments were passed.
func (a *Args) Values() []*Value {
	return a.args
}

func (a *Args) Named(name string) *Value {
	if len(a.kwArgs) > 0 {
		if v, exists := a.kwArgs[name]; exists {
			return v
		}
	}

	return emptyValue
}

func (a *Args) HasNamed(name string) bool {
	if len(a.kwArgs) > 0 {
		_, exists := a.kwArgs[name]
		return exists
	}

	return false
}

func (a *Args) NamedExists(name string) (*Value, bool) {
	if len(a.kwArgs) > 0 {
		if v, exists := a.kwArgs[name]; exists {
			return v, true
		}
	}
	return emptyValue, false
}

// Map returns the raw map of named arguments. This may be nil if no named arguments were passed.
func (a *Args) Map() map[string]*Value {
	return a.kwArgs
}

func (p *Parser) parseNamedAttribute() (string, *nodeFilteredVariable, error) {
	key := p.MatchType(TokenIdentifier)
	if p.Match(TokenSymbol, "=") == nil {
		// this should be impossible since we peeked to verify the '=' exists
		return "", nil, p.Error("expected '='", nil)
	}

	v, err := p.parseVariableOrLiteralWithFilter()
	if err != nil {
		return "", nil, err
	}

	return key.Val, v, nil
}

// Parses zero or more arguments; '(' should have already been consumed; will return without consuming
// the closing ')'.
func (p *Parser) parseArgs() (positional []IEvaluator, named map[string]IEvaluator, err error) {
	for {
		if p.Peek(TokenSymbol, ")") != nil {
			return
		} else if p.Peek(TokenSymbol, "}}") != nil {
			err = p.Error("Filter parameter or ')' required after '('.", nil)
			return
		}

		if p.PeekType(TokenIdentifier) != nil && p.PeekN(1, TokenSymbol, "=") != nil {
			k, v, e := p.parseNamedAttribute()
			if e != nil {
				err = e
				return
			}
			if named == nil {
				named = make(map[string]IEvaluator)
			}
			named[k] = v
		} else {
			v, e := p.parseVariableOrLiteralWithFilter()
			if e != nil {
				err = e
				return
			}

			if positional == nil {
				positional = make([]IEvaluator, 0, 4)
			}
			positional = append(positional, v)
		}

		if p.Match(TokenSymbol, ",") == nil {
			if p.Peek(TokenSymbol, ")") != nil {
				return
			}
			err = p.Error("',' or ')' expected", nil)
			return
		}
	}
}

var emptyArgs = &Args{}

func evaluateArgs(ctx *ExecutionContext, parameters []IEvaluator, namedParameters map[string]IEvaluator) (*Args, error) {
	var args *Args
	if len(parameters) > 0 {
		args = &Args{}
		args.args = make([]*Value, 0, len(parameters))
		for _, parameter := range parameters {
			param, err := parameter.Evaluate(ctx)
			if err != nil {
				return nil, err
			}
			args.args = append(args.args, param)
		}
	}
	if len(namedParameters) > 0 {
		if args == nil {
			args = &Args{}
		}
		args.kwArgs = make(map[string]*Value, len(namedParameters))
		for key, parameter := range namedParameters {
			param, err := parameter.Evaluate(ctx)
			if err != nil {
				return nil, err
			}
			args.kwArgs[key] = param
		}
	}

	if args == nil {
		args = emptyArgs
	}

	return args, nil
}

var ErrArgCount = errors.New("invalid parameter count")

// ExpectArgs asserts that the number of positional arguments is between min and max inclusive, otherwise it returns an
// Error. Only positional (not named) arguments are considered.
func ExpectArgs(typ, name string, min, max int, args *Args) error {
	argLen := args.Len()
	if argLen < min || (max != -1 && argLen > max) {
		var argRange string
		if min == max {
			argRange = strconv.Itoa(min)
		} else if max == -1 {
			argRange = fmt.Sprintf("at least %d", min)
		} else {
			argRange = fmt.Sprintf("%d-%d", min, max)
		}
		return &Error{
			Sender:    fmt.Sprintf("%s:%s", typ, name),
			OrigError: fmt.Errorf("%w: %s %s expected %s parameter(s), received %d", ErrArgCount, typ, name, argRange, argLen),
		}
	}
	return nil
}

var ErrArgName = errors.New("invalid parameter name")

// ExpectNamedArgs works similarly to ExpectArgs, but instead of specifying min/max, the required and optional arguments
// are specified by name. For example, required={"foo","bar","baz"} indicates that 3 positional arguments OR 0-3
// positional arguments supplemented by the remaining named arguments (such as f(1,bar=2,baz=3)) are required.
func ExpectNamedArgs(typ, name string, required []string, optional []string, args *Args) error {
	requiredCount := len(required)
	optionalCount := len(optional)
	totalCount := requiredCount + optionalCount

	if len(args.kwArgs) > 0 {
		for argName := range args.kwArgs {
			valid := false
			if len(required) > 0 {
				for _, name := range required {
					if name == argName {
						valid = true
						break
					}
				}
			}
			if !valid && len(optional) > 0 {
				for _, name := range optional {
					if name == argName {
						valid = true
						break
					}
				}
			}
			if !valid {
				return &Error{
					Sender:    fmt.Sprintf("%s:%s", typ, name),
					OrigError: fmt.Errorf("%w: %s", ErrArgName, argName),
				}
			}
		}
	}

	argLen := args.Len()

	invalid := false
	if argLen == requiredCount {
		return nil
	} else if argLen > totalCount {
		invalid = true
	} else if argLen < requiredCount {
		required = required[argLen:]
		for _, name := range required {
			if !args.HasNamed(name) {
				invalid = true
				break
			}
		}
	}

	if !invalid {
		return nil
	}

	var argRange string
	if requiredCount == totalCount {
		argRange = strconv.Itoa(requiredCount)
	} else {
		argRange = fmt.Sprintf("%d-%d", requiredCount, totalCount)
	}
	return &Error{
		Sender:    fmt.Sprintf("%s:%s", typ, name),
		OrigError: fmt.Errorf("%w: %s %s expected %s parameter(s), received %d", ErrArgCount, typ, name, argRange, argLen),
	}
}
