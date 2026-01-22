package pongo2

import (
	"reflect"
	"strings"
)

func init() {
	RegisterTest("callable", testCallable)
	RegisterTest("divisibleby", testDivisibleby)
	RegisterTest("eq", testEq)
	RegisterTest("==", testEq)
	RegisterTest("equalto", testEq)
	RegisterTest("escaped", testEscaped)
	RegisterTest("even", testEven)
	RegisterTest("false", testFalse)
	RegisterTest("falsy", testFalse)
	RegisterTest("filter", testFilter)
	RegisterTest("float", testFloat)
	RegisterTest("ge", testGe)
	RegisterTest(">=", testGe)
	RegisterTest("gt", testGt)
	RegisterTest("greaterthan", testGt)
	RegisterTest(">", testGt)
	RegisterTest("in", testIn)
	RegisterTest("integer", testInteger)
	RegisterTest("iterable", testIterable)
	RegisterTest("le", testLe)
	RegisterTest("<=", testLe)
	RegisterTest("lower", testLower)
	RegisterTest("lt", testLt)
	RegisterTest("<", testLt)
	RegisterTest("lessthan", testLt)
	RegisterTest("mapping", testMapping)
	RegisterTest("ne", testNe)
	RegisterTest("!=", testNe)
	RegisterTest("none", testNone)
	RegisterTest("number", testNumber)
	RegisterTest("odd", testOdd)
	RegisterTest("sameas", testSameas)
	RegisterTest("sequence", testSequence)
	RegisterTest("string", testString)
	RegisterTest("test", testTest)
	RegisterTest("true", testTrue)
	RegisterTest("truthy", testTrue)
	RegisterTest("upper", testUpper)
	RegisterTest("defined", testDefined)
	RegisterTest("undefined", testUndefined)
}

func testDefined(in *Value, args *Args) (bool, error) {
	return !in.IsNil(), nil
}

func testUndefined(in *Value, args *Args) (bool, error) {
	return in.IsNil(), nil
}

func testEscaped(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "safe", 0, 0, args); err != nil {
		return false, err
	}

	// TODO(sblinch): this isn't accurate; *Value.safe doesn't get updated when a Value is escaped, it just specifies
	// whether it was originally determined to be safe from needing escaping
	return in.IsNil(), nil
}

func testUpper(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "upper", 0, 0, args); err != nil {
		return false, err
	}

	switch {
	case in.IsString():
		s := in.String()
		return s == strings.ToUpper(s), nil
	default:
		return false, nil
	}
}

func testTrue(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "true", 0, 0, args); err != nil {
		return false, err
	}

	return in.IsTrue(), nil
}

func testTest(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "test", 0, 0, args); err != nil {
		return false, err
	}

	return TestExists(in.String()), nil
}

func testString(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "string", 0, 0, args); err != nil {
		return false, err
	}

	return in.IsString(), nil
}

func testSequence(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "sequence", 0, 0, args); err != nil {
		return false, err
	}

	return in.IsSliceOrArray() || in.IsString(), nil
}

func testSameas(in *Value, args *Args) (bool, error) {
	// TODO(sblinch): this is supposed to indicate whether the two values point to the same memory address
	return testEq(in, args)
}

func testOdd(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "odd", 0, 0, args); err != nil {
		return false, err
	}

	return in.Integer()%2 == 1, nil
}

func testNumber(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "number", 0, 0, args); err != nil {
		return false, err
	}

	return in.IsNumber(), nil
}

func testNone(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "none", 0, 0, args); err != nil {
		return false, err
	}

	return in.IsNil(), nil
}

func testNe(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "ne", 1, 1, args); err != nil {
		return false, err
	}

	return !reflect.DeepEqual(in.Interface(), args.First().Interface()), nil
}

func testMapping(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "mapping", 0, 0, args); err != nil {
		return false, err
	}

	return in.IsMap(), nil
}

func testLt(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "lt", 1, 1, args); err != nil {
		return false, err
	}

	return in.Compare(args.First()) == -1, nil

}

func testLower(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "lower", 0, 0, args); err != nil {
		return false, err
	}

	switch {
	case in.IsString():
		s := in.String()
		return s == strings.ToLower(s), nil
	default:
		return false, nil
	}
}

func testLe(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "le", 1, 1, args); err != nil {
		return false, err
	}

	return in.Compare(args.First()) != 1, nil
}

func testIterable(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "iterable", 0, 0, args); err != nil {
		return false, err
	}

	return in.IsIterable(), nil
}

func testInteger(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "integer", 0, 0, args); err != nil {
		return false, err
	}

	return in.IsInteger(), nil
}

func testIn(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "in", 1, 1, args); err != nil {
		return false, err
	}

	container := args.First()

	matched := false
	container.Iterate(func(idx, count int, key, value *Value) bool {
		v := value
		if value == nil {
			v = key
		}
		if v.EqualValueTo(in) {
			matched = true
			return false
		}

		return true
	}, nil)

	return matched, nil
}

func testGt(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "gt", 1, 1, args); err != nil {
		return false, err
	}

	return in.Compare(args.First()) == 1, nil
}

func testGe(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "ge", 1, 1, args); err != nil {
		return false, err
	}

	return in.Compare(args.First()) != -1, nil
}

func testFloat(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "float", 0, 0, args); err != nil {
		return false, err
	}

	return in.IsFloat(), nil
}

func testFilter(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "filter", 0, 0, args); err != nil {
		return false, err
	}

	return BuiltinFilterExists(in.String()), nil
}

func testFalse(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "false", 0, 0, args); err != nil {
		return false, err
	}

	return !in.IsTrue(), nil
}

func testEven(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "even", 0, 0, args); err != nil {
		return false, err
	}

	return in.Integer()%2 == 0, nil
}

func testEq(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "eq", 1, 1, args); err != nil {
		return false, err
	}

	return reflect.DeepEqual(in.Interface(), args.First().Interface()), nil
}

func testDivisibleby(in *Value, args *Args) (bool, error) {
	if err := ExpectArgs("test", "divisibleby", 1, 1, args); err != nil {
		return false, err
	}

	vIn := in.Integer()
	vOut := args.First().Integer()
	if vOut == 0 {
		return false, nil
	}
	return vIn%vOut == 0, nil
}

func testCallable(in *Value, args *Args) (bool, error) {
	// placeholder; implemented internally in testCall.Evaluate
	return false, nil
}
