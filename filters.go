package pongo2

import (
	"fmt"
	"maps"
)

// FilterFunction is the type filter functions must fulfil
type FilterFunction func(in *Value, param *Value) (out *Value, err error)

// FilterArgsFunction is the type filter functions supporting arguments must fulfil
type FilterArgsFunction func(in *Value, args *Args) (out *Value, err error)

var builtinFilters = make(map[string]FilterFunction)
var builtinFilterArgs = make(map[string]FilterArgsFunction)

func wrapFilterFunc(fn FilterFunction) FilterArgsFunction {
	return func(in *Value, args *Args) (out *Value, err error) {
		return fn(in, args.First())
	}
}

// copyFilters creates a shallow copy of a filter map.
func copyFilters(src map[string]FilterFunction) map[string]FilterFunction {
	dst := make(map[string]FilterFunction, len(src))
	maps.Copy(dst, src)
	return dst
}

// copyFilterArgs creates a shallow copy of a filterArgs map.
func copyFilterArgs(src map[string]FilterArgsFunction) map[string]FilterArgsFunction {
	dst := make(map[string]FilterArgsFunction, len(src))
	maps.Copy(dst, src)
	return dst
}

// BuiltinFilterExists returns true if the given filter is a built-in filter.
// Use TemplateSet.FilterExists to check filters in a specific template set.
func BuiltinFilterExists(name string) bool {
	_, existing := builtinFilters[name]
	if !existing {
		_, existing = builtinFilterArgs[name]
	}
	return existing
}

// BuiltinTagExists returns true if the given tag is registered in builtinTags.
// Use TemplateSet.TagExists to check tags in a specific template set.
func BuiltinTagExists(name string) bool {
	_, existing := builtinTags[name]
	return existing
}

// registerFilterBuiltin registers a new filter to the global filter map.
// This is used during package initialization to register builtin filters.
func registerFilterBuiltin(name string, fn FilterFunction) error {
	if BuiltinFilterExists(name) {
		return fmt.Errorf("filter with name '%s' is already registered", name)
	}
	builtinFilters[name] = fn
	return nil
}

// registerFilterArgsBuiltin registers a new filter with args to the global filter map.
// This is used during package initialization to register builtin filters.
func registerFilterArgsBuiltin(name string, fn FilterArgsFunction) error {
	if _, exists := builtinFilterArgs[name]; exists {
		return fmt.Errorf("filter with name '%s' is already registered", name)
	}
	delete(builtinFilters, name)
	builtinFilterArgs[name] = fn
	return nil
}

func AliasBuiltinFilter(name, alias string) error {
	if !BuiltinFilterExists(name) {
		return fmt.Errorf("filter with name '%s' does not exist (therefore cannot be aliased)", name)
	}
	if _, exists := builtinFilters[name]; exists {
		builtinFilters[alias] = builtinFilters[name]
	} else {
		builtinFilterArgs[alias] = builtinFilterArgs[name]
	}
	return nil
}

// MustApplyFilter behaves like ApplyFilter, but panics on an error.
// This function uses builtinFilters. Use TemplateSet.MustApplyFilter for set-specific filters.
func MustApplyFilter(name string, value *Value, param *Value) *Value {
	val, err := ApplyFilter(name, value, param)
	if err != nil {
		panic(err)
	}
	return val
}

// ApplyFilter applies a built-infilter to a given value using the given
// parameters. Returns a *pongo2.Value or an error. Use TemplateSet.ApplyFilter
// for set-specific filters.
func ApplyFilter(name string, value *Value, param *Value) (*Value, error) {
	fn, existing := builtinFilters[name]
	if !existing {
		if fan, existing := builtinFilterArgs[name]; existing {
			return fan(value, NewArgs(nil, param))
		}

		return nil, &Error{
			Sender:    "applyfilter",
			OrigError: fmt.Errorf("filter with name '%s' not found", name),
		}
	}

	// Make sure param is a *Value
	if param == nil {
		param = AsValue(nil)
	}

	return fn(value, param)
}

// ApplyFilterArgs applies a built-infilter to a given value using the given
// parameters. Returns a *pongo2.Value or an error. Use TemplateSet.ApplyFilterArgs
// for set-specific filters.
func ApplyFilterArgs(name string, value *Value, args *Args) (*Value, error) {
	fn, existing := builtinFilterArgs[name]
	if !existing {
		if len(args.args)+len(args.kwArgs) < 2 {
			if f, existing := builtinFilters[name]; existing {
				var param *Value
				if args.Len() > 0 {
					param = args.Value(0)
				} else if len(args.kwArgs) > 0 {
					for _, v := range args.kwArgs {
						param = v
					}
				}
				return f(value, param)
			}
		}

		return nil, &Error{
			Sender:    "applyfilter",
			OrigError: fmt.Errorf("filter with name '%s' not found", name),
		}
	}

	if args == nil {
		args = NewArgs(nil, nil)
	}

	return fn(value, args)
}

type filterCall struct {
	token *Token

	name string

	parameter  IEvaluator
	filterFunc FilterFunction

	parameters      []IEvaluator
	namedParameters map[string]IEvaluator
	filterArgsFunc  FilterArgsFunction
}

func (fc *filterCall) Execute(v *Value, ctx *ExecutionContext) (*Value, error) {
	var (
		filteredValue *Value
		err           error
	)
	if fc.filterFunc != nil {
		var param *Value

		if fc.parameter != nil {
			param, err = fc.parameter.Evaluate(ctx)
			if err != nil {
				return nil, err
			}
		} else {
			param = AsValue(nil)
		}

		filteredValue, err = fc.filterFunc(v, param)
	} else {
		var args *Args
		args, err = evaluateArgs(ctx, fc.parameters, fc.namedParameters)
		if err != nil {
			return nil, err
		}
		filteredValue, err = fc.filterArgsFunc(v, args)
	}
	if err != nil {
		return nil, updateErrorToken(err, ctx.template, fc.token)
	}
	return filteredValue, nil
}

// Filter = IDENT | IDENT ":" FilterArg | IDENT "|" Filter
func (p *Parser) parseFilter() (*filterCall, error) {
	identToken := p.MatchType(TokenIdentifier)

	// Check filter ident
	if identToken == nil {
		return nil, p.Error("Filter name must be an identifier.", nil)
	}

	filter := &filterCall{
		token: identToken,
		name:  identToken.Val,
	}

	// Check sandbox filter restriction
	if _, isBanned := p.template.set.bannedFilters[identToken.Val]; isBanned {
		return nil, p.Error(fmt.Sprintf("Usage of filter '%s' is not allowed (sandbox restriction active).", identToken.Val), identToken)
	}

	// Get the appropriate filter function and bind it
	filterFn, exists := p.template.set.filters[identToken.Val]
	if exists {
		filter.filterFunc = filterFn
	} else {
		filterArgsFn, exists := p.template.set.filterArgs[identToken.Val]
		if !exists {
			return nil, p.Error(fmt.Sprintf("Filter '%s' does not exist.", identToken.Val), identToken)
		}
		filter.filterArgsFunc = filterArgsFn
	}

	// Check for filter-argument (2 tokens needed: ':' ARG)
	if p.Match(TokenSymbol, ":") != nil {
		if p.Peek(TokenSymbol, "}}") != nil {
			return nil, p.Error("Filter parameter required after ':'.", nil)
		}

		// Get filter argument expression
		v, err := p.parseVariableOrLiteral()
		if err != nil {
			return nil, err
		}

		if filter.filterFunc != nil {
			filter.parameter = v
		} else {
			filter.parameters = append(filter.parameters, v)
		}

	} else if p.Match(TokenSymbol, "(") != nil {
		var err error
		filter.parameters, filter.namedParameters, err = p.parseArgs()
		if err != nil {
			return nil, err
		}

		if p.Match(TokenSymbol, ")") == nil {
			return nil, p.Error("')' expected", nil)
		}

		if filter.filterArgsFunc == nil {
			// if this filter was registered with Django (single argument string) syntax, we can only call it if the
			// template passed less than 2 parameters
			if len(filter.parameters)+len(filter.namedParameters) > 1 {
				return nil, p.Error("Too many parameters for this filter call.", nil)

			} else if len(filter.parameters) > 0 {
				filter.parameter = filter.parameters[0]

			} else {
				for _, param := range filter.namedParameters {
					filter.parameter = param
				}
			}
		}

	}

	return filter, nil
}
