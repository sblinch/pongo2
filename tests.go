package pongo2

import (
	"fmt"
	"reflect"
)

// TestFunction is the type test functions must fulfil
type TestFunction func(in *Value, args *Args) (bool, error)

var tests map[string]TestFunction

func init() {
	tests = make(map[string]TestFunction)
}

// TestExists returns true if the given test is already registered
func TestExists(name string) bool {
	_, existing := tests[name]
	return existing
}

// RegisterTest registers a new test. If there's already a test with the same
// name, RegisterTest will panic. You usually want to call this
// function in the test's init() function:
// http://golang.org/doc/effective_go.html#init
func RegisterTest(name string, fn TestFunction) error {
	if TestExists(name) {
		return fmt.Errorf("test with name '%s' is already registered", name)
	}
	tests[name] = fn
	return nil
}

// ReplaceTest replaces an already registered test with a new implementation. Use this
// function with caution since it allows you to change existing test behaviour.
func ReplaceTest(name string, fn TestFunction) error {
	if !TestExists(name) {
		return fmt.Errorf("test with name '%s' does not exist (therefore cannot be overridden)", name)
	}
	tests[name] = fn
	return nil
}

// MustPerformTest behaves like PerformTest, but panics on an error.
func MustPerformTest(name string, value *Value, args *Args) bool {
	val, err := PerformTest(name, value, args)
	if err != nil {
		panic(err)
	}
	return val
}

// PerformTest performs a test on a given value using the given parameters.
// Returns a bool or an error.
func PerformTest(name string, value *Value, args *Args) (bool, error) {
	fn, existing := tests[name]
	if !existing {
		return false, &Error{
			Sender:    "performtest",
			OrigError: fmt.Errorf("test with name '%s' not found", name),
		}
	}

	return fn(value, args)
}

type testCall struct {
	token *Token

	name            string
	parameters      []IEvaluator
	namedParameters map[string]IEvaluator

	term IEvaluator

	negate   bool
	testFunc TestFunction
}

func (expr *testCall) FilterApplied(name string) bool {
	return expr.term.FilterApplied(name)
}

func (expr *testCall) GetPositionToken() *Token {
	return expr.term.GetPositionToken()
}

func (expr *testCall) Execute(ctx *ExecutionContext, writer TemplateWriter) error {
	value, err := expr.Evaluate(ctx)
	if err != nil {
		return err
	}
	writer.WriteString(value.String())
	return nil
}

func (tc *testCall) Evaluate(ctx *ExecutionContext) (*Value, error) {
	passed := false
	undefined := false
	switch tc.name {
	case "undefined":
		// undefined and defined are implemented here because they require access to ctx
		undefined = true
		fallthrough
	case "defined":
		resolved, err := tc.term.Evaluate(ctx)
		passed = err == nil && resolved.getResolvedValue().IsValid()
		if undefined {
			passed = !passed
		}
	case "escaped":
		// escaped is implemented here because it requires access to tc and ctx
		passed = tc.FilterApplied("safe") || tc.term.FilterApplied("escape")
		if !passed {
			if vv, err := tc.term.Evaluate(ctx); err == nil {
				passed = vv.safe
			}
		}

	case "callable":
		// callable requires a custom implementation as variableResolver.Evaluate normally invokes referenced functions
		// even if they aren't followed by (); this is incompatible with checking whether a variable is a function
		if nfv, ok := tc.term.(*nodeFilteredVariable); ok {
			if vr, ok := nfv.resolver.(*variableResolver); ok {
				tmpvr := &variableResolver{}
				*tmpvr = *vr
				tmpvr.parts = append(make([]*variablePart, 0, len(vr.parts)), vr.parts...)
				// evaluate with i=1 as a flag to obtain the Func itself rather than its return value
				tmpvr.parts[len(tmpvr.parts)-1].i = 1

				f, err := vr.Evaluate(ctx)
				if err != nil {
					return AsValue(false), nil
				}
				return AsValue(f.getResolvedValue().Kind() == reflect.Func), nil
			}

		}
		return AsValue(false), nil

	default:
		t, err := tc.term.Evaluate(ctx)
		if err != nil {
			return nil, err
		}

		args, err := evaluateArgs(ctx, tc.parameters, tc.namedParameters)
		if err != nil {
			return nil, err
		}

		passed, err = PerformTest(tc.name, t, args)
		if err != nil {
			if e, ok := err.(*Error); ok {
				err = e.updateFromTokenIfNeeded(ctx.template, tc.token)
			}
			return nil, err
		}

	}
	if tc.negate {
		passed = !passed
	}
	return AsValue(passed), nil
}

func (p *Parser) parseTest(term IEvaluator) (IEvaluator, error) {
	negate := false
	if t := p.MatchOne(TokenKeyword, "not"); t != nil {
		negate = true
	}

	identToken := p.MatchType(TokenIdentifier)
	if identToken == nil {
		// allow ==, >=, etc as test names
		identToken = p.MatchType(TokenSymbol)
	}
	if identToken == nil {
		// allow true/false as test names
		identToken = p.MatchType(TokenKeyword)
	}

	// Check filter ident
	if identToken == nil {
		return nil, p.Error("Test name must be an identifier.", nil)
	}

	test := &testCall{
		token:  identToken,
		name:   identToken.Val,
		term:   term,
		negate: negate,
	}

	// Value the appropriate tests function and bind it
	testFn, exists := tests[identToken.Val]
	if !exists {
		return nil, p.Error(fmt.Sprintf("Test '%s' does not exist.", identToken.Val), identToken)
	}

	test.testFunc = testFn

	if p.Match(TokenSymbol, "(") != nil {
		var err error
		test.parameters, test.namedParameters, err = p.parseArgs()
		if err != nil {
			return nil, err
		}

		if p.Match(TokenSymbol, ")") == nil {
			return nil, p.Error("')' expected", nil)
		}

	} else if p.PeekType(TokenIdentifier) != nil || p.PeekType(TokenString) != nil || p.PeekType(TokenNumber) != nil || p.PeekType(TokenNil) != nil {
		arg, err := p.parseVariableOrLiteral()
		if err != nil {
			return nil, err
		}
		test.parameters = []IEvaluator{arg}
	}

	return test, nil
}
