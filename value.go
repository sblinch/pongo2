package pongo2

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Value struct {
	val  reflect.Value
	safe bool // used to indicate whether a Value needs explicit escaping in the template
}

// AsValue converts any given value to a pongo2.Value
// Usually being used within own functions passed to a template
// through a Context or within filter functions.
//
// Example:
//
//	AsValue("my string")
func AsValue(i any) *Value {
	return &Value{
		val: reflect.ValueOf(i),
	}
}

// AsSafeValue works like AsValue, but does not apply the 'escape' filter.
func AsSafeValue(i any) *Value {
	return &Value{
		val:  reflect.ValueOf(i),
		safe: true,
	}
}

func (v *Value) getResolvedValue() reflect.Value {
	rv := v.val
	// Unwrap pointers and interfaces to get to the underlying value
	for rv.IsValid() && (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) {
		rv = rv.Elem()
	}
	return rv
}

// IsString checks whether the underlying value is a string
func (v *Value) IsString() bool {
	return v.getResolvedValue().Kind() == reflect.String || v.IsTemplate()
}

// IsBool checks whether the underlying value is a bool
func (v *Value) IsBool() bool {
	return v.getResolvedValue().Kind() == reflect.Bool
}

// IsFloat checks whether the underlying value is a float
func (v *Value) IsFloat() bool {
	kind := v.getResolvedValue().Kind()
	return kind == reflect.Float32 || kind == reflect.Float64
}

// IsInteger checks whether the underlying value is an integer
func (v *Value) IsInteger() bool {
	kind := v.getResolvedValue().Kind()
	return kind == reflect.Int ||
		kind == reflect.Int8 ||
		kind == reflect.Int16 ||
		kind == reflect.Int32 ||
		kind == reflect.Int64 ||
		kind == reflect.Uint ||
		kind == reflect.Uint8 ||
		kind == reflect.Uint16 ||
		kind == reflect.Uint32 ||
		kind == reflect.Uint64
}

func (v *Value) Is64BitInteger() bool {
	kind := v.getResolvedValue().Kind()
	return kind == reflect.Int64 || kind == reflect.Uint64
}

// IsNumber checks whether the underlying value is either an integer
// or a float.
func (v *Value) IsNumber() bool {
	return v.IsInteger() || v.IsFloat()
}

// IsTime checks whether the underlying value is a time.Time.
func (v *Value) IsTime() bool {
	_, ok := v.Interface().(time.Time)
	return ok
}

// IsNil checks whether the underlying value is NIL
func (v *Value) IsNil() bool {
	// fmt.Printf("%+v\n", v.getResolvedValue().Type().String())
	return !v.getResolvedValue().IsValid()
}

func (v *Value) IsScalar() bool {
	kind := v.getResolvedValue().Kind()
	switch kind {
	case reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		return true
	case reflect.Struct:
		return v.IsTemplate()
	}
	return false
}

func (v *Value) IsTemplate() bool {
	rv := v.getResolvedValue()
	if rv.Kind() == reflect.Struct {
		_, isTemplate := rv.Interface().(Template)
		return isTemplate
	}
	return false
}

func (v *Value) EvaluateTemplate(ctx Context) (*Value, error) {
	val := v.val
	var (
		prev reflect.Value
	)
	for val.IsValid() {
		switch val.Kind() {
		case reflect.Interface, reflect.Ptr:
			prev = val
			val = val.Elem()
		case reflect.Struct:
			tpl, isTemplate := prev.Interface().(*Template)
			if !isTemplate {
				return AsValue(nil), nil
			}
			evaluated, err := tpl.Evaluate(ctx)
			if err != nil {
				return nil, err
			}
			return AsValue(evaluated), nil
		default:
			return AsValue(nil), nil
		}
	}

	return AsValue(nil), nil
}

// String returns a string for the underlying value. If this value is not
// of type string, pongo2 tries to convert it. Currently the following
// types for underlying values are supported:
//
//  1. string
//  2. int/uint (any size)
//  3. float (any precision)
//  4. bool
//  5. time.Time
//  6. String() will be called on the underlying value if provided
//
// NIL values will lead to an empty string. Unsupported types are leading
// to their respective type name.
func (v *Value) String() string {
	if v.IsNil() {
		return ""
	}

	if t, ok := v.Interface().(fmt.Stringer); ok {
		return t.String()
	}

	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.String:
		return rv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", rv.Float())
	case reflect.Bool:
		if v.Bool() {
			return "True"
		}
		return "False"
	}

	logf("Value.String() not implemented for type: %s\n", rv.Kind().String())
	return rv.String()
}

// Integer returns the underlying value as an integer (converts the underlying
// value, if necessary). If it's not possible to convert the underlying value,
// it will return 0.
func (v *Value) Integer() int {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := rv.Uint()
		if u > math.MaxInt {
			return math.MaxInt
		}
		return int(u)
	case reflect.Float32, reflect.Float64:
		return int(rv.Float())
	case reflect.String:
		s := rv.String()
		if len(s) > 1 && s[0] == '0' {
			switch s[1] {
			case 'x':
				i, _ := strconv.ParseInt(s[2:], 16, 64)
				return int(i)
			case 'b':
				i, _ := strconv.ParseInt(s[2:], 2, 64)
				return int(i)
			case 'o', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				i, _ := strconv.ParseInt(s[2:], 8, 64)
				return int(i)
			}
		}
		// Try to convert from string to int (base 10)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return int(f)
	default:
		logf("Value.Integer() not available for type: %s\n", rv.Kind().String())
		return 0
	}
}

// Int64 is like Integer but returns an int64
func (v *Value) Int64() int64 {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := rv.Uint()
		if u > math.MaxInt64 {
			return math.MaxInt64
		}
		return int64(u)
	case reflect.Float32, reflect.Float64:
		return int64(rv.Float())
	case reflect.String:
		s := rv.String()
		if len(s) > 1 && s[0] == '0' {
			switch s[1] {
			case 'x':
				i, _ := strconv.ParseInt(s[2:], 16, 64)
				return i
			case 'b':
				i, _ := strconv.ParseInt(s[2:], 2, 64)
				return i
			case 'o', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				i, _ := strconv.ParseInt(s[2:], 8, 64)
				return i
			}
		}
		// Try to convert from string to int (base 10)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return int64(f)
	default:
		logf("Value.Int64() not available for type: %s\n", rv.Kind().String())
		return 0
	}
}

// Float returns the underlying value as a float (converts the underlying
// value, if necessary). If it's not possible to convert the underlying value,
// it will return 0.0.
func (v *Value) Float() float64 {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	case reflect.String:
		// Try to convert from string to float64 (base 10)
		f, err := strconv.ParseFloat(rv.String(), 64)
		if err != nil {
			return 0.0
		}
		return f
	default:
		logf("Value.Float() not available for type: %s\n", rv.Kind().String())
		return 0.0
	}
}

// Bool returns the underlying value as bool. If the value is not bool, false
// will always be returned. If you're looking for true/false-evaluation of the
// underlying value, have a look on the IsTrue()-function.
func (v *Value) Bool() bool {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Bool:
		return rv.Bool()
	default:
		logf("Value.Bool() not available for type: %s\n", rv.Kind().String())
		return false
	}
}

// Time returns the underlying value as time.Time.
// If the underlying value is not a time.Time, it returns the zero value of time.Time.
func (v *Value) Time() time.Time {
	tm, ok := v.Interface().(time.Time)
	if ok {
		return tm
	}
	return time.Time{}
}

// IsTrue tries to evaluate the underlying value the Pythonic-way:
//
// Returns TRUE in one the following cases:
//
//   - int != 0
//   - uint != 0
//   - float != 0.0
//   - len(array/chan/map/slice/string) > 0
//   - bool == true
//   - underlying value is a struct
//
// Otherwise returns always FALSE.
func (v *Value) IsTrue() bool {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() != 0
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return rv.Len() > 0
	case reflect.Bool:
		return rv.Bool()
	case reflect.Struct:
		return true // struct instance is always true
	default:
		logf("Value.IsTrue() not available for type: %s\n", rv.Kind().String())
		return false
	}
}

// Negate tries to negate the underlying value. It's mainly used for
// the NOT-operator and in conjunction with a call to
// return_value.IsTrue() afterwards.
//
// Example:
//
//	AsValue(1).Negate().IsTrue() == false
func (v *Value) Negate() *Value {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.Integer() != 0 {
			return AsValue(0)
		}
		return AsValue(1)
	case reflect.Float32, reflect.Float64:
		if v.Float() != 0.0 {
			return AsValue(float64(0.0))
		}
		return AsValue(float64(1.1))
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return AsValue(rv.Len() == 0)
	case reflect.Bool:
		return AsValue(!rv.Bool())
	case reflect.Struct:
		return AsValue(false)
	default:
		logf("Value.IsTrue() not available for type: %s\n", rv.Kind().String())
		return AsValue(true)
	}
}

// Len returns the length for an array, chan, map, slice or string.
// Otherwise it will return 0.
func (v *Value) Len() int {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return rv.Len()
	case reflect.String:
		runes := []rune(rv.String())
		return len(runes)
	default:
		logf("Value.Len() not available for type: %s\n", rv.Kind().String())
		return 0
	}
}

// Slice slices an array, slice or string. Otherwise it will
// return an empty []int.
func (v *Value) Slice(i, j int) *Value {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		return AsValue(rv.Slice(i, j).Interface())
	case reflect.String:
		runes := []rune(rv.String())
		return AsValue(string(runes[i:j]))
	default:
		logf("Value.Slice() not available for type: %s\n", rv.Kind().String())
		return AsValue([]int{})
	}
}

// Index gets the i-th item of an array, slice or string. Otherwise
// it will return NIL.
func (v *Value) Index(i int) *Value {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		if i >= v.Len() {
			return AsValue(nil)
		}
		return AsValue(rv.Index(i).Interface())
	case reflect.String:
		runes := []rune(rv.String())
		if i < len(runes) {
			return AsValue(string(runes[i]))
		}
		return AsValue("")
	default:
		logf("Value.Slice() not available for type: %s\n", rv.Kind().String())
		return AsValue([]int{})
	}
}

// SetIndex sets the i-th item of an array or slice. Will panic if the underlying value of val does not match the
// element type of the array or slice.
func (v *Value) SetIndex(i int, val *Value) {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		if i < v.Len() {
			rv.Index(i).Set(val.getResolvedValue())
		}
	default:
		logf("Value.SetIndex() not available for type: %s\n", rv.Kind().String())
	}
}

// Contains checks whether the underlying value (which must be of type struct, map,
// string, array or slice) contains of another Value (e. g. used to check
// whether a struct contains of a specific field or a map contains a specific key).
//
// Example:
//
//	AsValue("Hello, World!").Contains(AsValue("World")) == true
func (v *Value) Contains(other *Value) bool {
	baseValue := v.getResolvedValue()
	switch baseValue.Kind() {
	case reflect.Struct:
		fieldValue := baseValue.FieldByName(other.String())
		return fieldValue.IsValid()
	case reflect.Map:
		// We can't check against invalid types
		if !other.val.IsValid() {
			return false
		}
		// Ensure that map key type is equal to other's type.
		if baseValue.Type().Key() != other.val.Type() {
			return false
		}

		// Use MapIndex directly - type check already verified key type matches
		mapValue := baseValue.MapIndex(other.getResolvedValue())
		return mapValue.IsValid()
	case reflect.String:
		return strings.Contains(baseValue.String(), other.String())

	case reflect.Slice, reflect.Array:
		for i := 0; i < baseValue.Len(); i++ {
			item := baseValue.Index(i)
			if other.EqualValueTo(AsValue(item.Interface())) {
				return true
			}
		}
		return false

	default:
		logf("Value.Contains() not available for type: %s\n", baseValue.Kind().String())
		return false
	}
}

// CanSlice checks whether the underlying value is of type array, slice or string.
// You normally would use CanSlice() before using the Slice() operation.
func (v *Value) CanSlice() bool {
	switch v.getResolvedValue().Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
		return true
	}
	return false
}

// IsSliceOrArray returns true if the value is a slice or array (not a string)
func (v *Value) IsSliceOrArray() bool {
	switch v.getResolvedValue().Kind() {
	case reflect.Array, reflect.Slice:
		return true
	}
	return false
}

// IsIterable checks whether the underlying value is iterable
func (v *Value) IsIterable() bool {
	switch v.getResolvedValue().Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return true
	}
	return false
}

// IsMap checks whether the underlying value is a map
func (v *Value) IsMap() bool {
	return v.getResolvedValue().Kind() == reflect.Map
}

// IsStruct checks whether the underlying value is a struct
func (v *Value) IsStruct() bool {
	return v.getResolvedValue().Kind() == reflect.Struct
}

// Element returns a Value of the key name from a map[string]T. If the underlying value is not a map, an empty value
// will be returned.
func (v *Value) Element(name string) *Value {
	rv := v.getResolvedValue()
	for rv.Kind() == reflect.Interface && !rv.IsNil() {
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Map:
		v := rv.MapIndex(reflect.ValueOf(name))
		return &Value{val: v}
	}
	return &Value{}
}

func (v *Value) SetElement(name string, value *Value) {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Map:
		//		println("set element",name,"of",v.String(),"to",value.String())
		rv.SetMapIndex(reflect.ValueOf(name), value.getResolvedValue())
	}
}

// NestedElement is similar to Element, but accepts a string slice of strings representing a list of nested keys. For
// example, names={"foo","bar"} would attempt to return v.Element("foo").Element("bar").
func (v *Value) NestedElement(names []string) *Value {
	r := v
	for len(names) > 0 {
		if !r.IsMap() {
			return &Value{}
		}
		r = r.Element(names[0])
		names = names[1:]
	}

	return r
}

// Attribute returns the specified map attribute if the underlying value is a map. Dot-separated values are supported
// to access nested keys, eg: Attribute("foo") == map["foo"] and Attribute("foo.bar") == map["foo"]["bar"].
func (v *Value) Attribute(attribute string) *Value {
	return v.NestedElement(strings.Split(attribute, "."))
}

// GetItem retrieves a value from a map by key or a field from a struct by name.
// For maps, it attempts to convert the key to the map's key type.
// For structs, it uses the key's string representation as the field name.
// Returns nil Value if the key/field doesn't exist or the type doesn't support item access.
func (v *Value) GetItem(key *Value) *Value {
	if key.IsNil() {
		return AsValue(nil)
	}

	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Map:
		keyStr := key.String()
		mapKeyType := rv.Type().Key()

		// Try to get the map value using appropriate key type
		var mapKey reflect.Value
		switch mapKeyType.Kind() {
		case reflect.String:
			mapKey = reflect.ValueOf(keyStr)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			mapKey = reflect.ValueOf(key.Integer()).Convert(mapKeyType)
		default:
			// Try direct conversion if the key type matches
			if key.val.IsValid() && key.val.Type().ConvertibleTo(mapKeyType) {
				mapKey = key.val.Convert(mapKeyType)
			} else {
				return AsValue(nil)
			}
		}

		val := rv.MapIndex(mapKey)
		if val.IsValid() {
			return &Value{val: val}
		}
		return AsValue(nil)

	case reflect.Struct:
		field := rv.FieldByName(key.String())
		if field.IsValid() {
			return &Value{val: field}
		}
		return AsValue(nil)

	default:
		return AsValue(nil)
	}
}

// Iterate iterates over a map, array, slice or a string. It calls the
// function's first argument for every value with the following arguments:
//
//	idx      current 0-index
//	count    total amount of items
//	key      *Value for the key or item
//	value    *Value (only for maps, the respective value for a specific key)
//
// If the underlying value has no items or is not one of the types above,
// the empty function (function's second argument) will be called.
func (v *Value) Iterate(fn func(idx, count int, key, value *Value) bool, empty func()) {
	v.IterateOrder(fn, empty, false, false)
}

// IterateOrder behaves like Value.Iterate, but can iterate through an array/slice/string in reverse. Does
// not affect the iteration through a map because maps don't have any particular order.
// However, you can force an order using the `sorted` keyword (and even use `reversed sorted`).
func (v *Value) IterateOrder(fn func(idx, count int, key, value *Value) bool, empty func(), reverse bool, sorted bool) {
	rv := v.getResolvedValue()
	switch rv.Kind() {
	case reflect.Map:
		keys := sortedKeys(rv.MapKeys())
		if sorted {
			if reverse {
				sort.Sort(sort.Reverse(keys))
			} else {
				sort.Sort(keys)
			}
		}
		keyLen := len(keys)
		for idx, key := range keys {
			value := rv.MapIndex(key)
			if !fn(idx, keyLen, &Value{val: key}, &Value{val: value}) {
				return
			}
		}
		if keyLen == 0 {
			empty()
		}
		return // done
	case reflect.Array, reflect.Slice:
		var items valuesList

		itemCount := rv.Len()
		for i := range itemCount {
			items = append(items, &Value{val: rv.Index(i)})
		}

		if sorted {
			if reverse {
				sort.Sort(sort.Reverse(items))
			} else {
				sort.Sort(items)
			}
		} else {
			if reverse {
				for i := 0; i < itemCount/2; i++ {
					items[i], items[itemCount-1-i] = items[itemCount-1-i], items[i]
				}
			}
		}

		if len(items) > 0 {
			for idx, item := range items {
				if !fn(idx, itemCount, item, nil) {
					return
				}
			}
		} else {
			empty()
		}
		return // done
	case reflect.String:
		rs := []rune(rv.String())
		charCount := len(rs)

		if charCount > 0 {
			if sorted {
				sort.SliceStable(rs, func(i, j int) bool {
					return rs[i] < rs[j]
				})
			}

			if reverse {
				for i, j := 0, charCount-1; i < j; i, j = i+1, j-1 {
					rs[i], rs[j] = rs[j], rs[i]
				}
			}

			for i := range charCount {
				if !fn(i, charCount, &Value{val: reflect.ValueOf(string(rs[i]))}, nil) {
					return
				}
			}
		} else {
			empty()
		}
		return // done
	default:
		logf("Value.Iterate() not available for type: %s\n", rv.Kind().String())
	}
	empty()
}

// MapFunc is the function called by Map() for each item in the Iterable. Note that unlike Iterate which (for backward
// compatibility) provides the item value as `key` when iterating over slices, Map always provides the item value as
// `value` for maps, slices, and strings. For slices and strings, `key` is nil.
type MapFunc func(idx, count int, key, value *Value) (newKey, newValue *Value, err error)

// Map is similar to Iterate, but it returns a new Value of the same kind as the underlying value which contains the
// Values returned by fn. If fn returns (nil, nil, nil) for any item, it will be omitted. If fn returns a non-nil Error,
// Map will abort and return the same Error with a nil Value.
//
// The types returned are:
// - If the underlying value is a Map: map[string]interface{}
// - If a slice: []interface{}
// - If a string: string if all returned Values are strings, bytes, or runes; otherwise []interface{}
// - Otherwise: nil
func (v *Value) Map(fn MapFunc) (*Value, error) {
	switch v.getResolvedValue().Kind() {
	case reflect.Map:
		keys := sortedKeys(v.getResolvedValue().MapKeys())
		keyLen := len(keys)
		r := make(map[string]interface{}, keyLen)
		for idx, key := range keys {
			value := v.getResolvedValue().MapIndex(key)
			newKey, newValue, err := fn(idx, keyLen, &Value{val: key}, &Value{val: value})
			if err != nil {
				return nil, err
			}
			if newKey != nil && newValue != nil {
				r[newKey.String()] = newValue.Interface()
			}
		}
		return AsValue(r), nil
	case reflect.Array, reflect.Slice:
		itemCount := v.getResolvedValue().Len()
		r := make([]interface{}, 0, itemCount)
		for idx := 0; idx < itemCount; idx++ {
			_, newItem, err := fn(idx, itemCount, nil, &Value{val: v.getResolvedValue().Index(idx)})
			if err != nil {
				return nil, err
			}
			if newItem != nil {
				r = append(r, newItem.Interface())
			}
		}
		return AsValue(r), nil
	case reflect.String:
		inputString := v.getResolvedValue().String()
		charCount := utf8.RuneCountInString(inputString)

		var (
			b strings.Builder
			r []interface{}
		)
		toString := true

	stringAgain:
		for i, c := range inputString {
			newC, _, err := fn(i, charCount, nil, &Value{val: reflect.ValueOf(c)})
			if err != nil {
				return nil, err
			}
			if newC == nil {
				continue
			}

			if toString {
				switch newC.getResolvedValue().Kind() {
				case reflect.Int8, reflect.Int32:
					// we can't know for certain that the map function is returning byte/rune values in this case, but it's
					// probably a fair bet
					b.WriteRune(rune(newC.Integer()))
				case reflect.String:
					b.WriteString(newC.String())
				default:
					toString = false
					r = make([]interface{}, 0, charCount)
					goto stringAgain
				}
			} else {
				r = append(r, newC.Interface())
			}
		}

		if toString {
			return AsValue(b.String()), nil
		} else {
			return AsValue(r), nil
		}
	default:
		//		return nil, &Error{OrigError: fmt.Errorf("cannot apply map function to value of type %s", v.getResolvedValue().Kind().String())}
		return AsValue(nil), nil
	}
}

// ShallowCopy makes a shallow copy of the underlying value. If the underlying value is wrapped in pointer or
// interface{} values, those layers of indirection are removed in the returned value; otherwise the returned value is of
// the same type as the original.
func (v *Value) ShallowCopy() *Value {
	var r reflect.Value
	rv := v.getResolvedValue()
	wasPtr := v.val.Kind() == reflect.Ptr

	if v.IsMap() {
		r = reflect.MakeMapWithSize(rv.Type(), v.Len())
		for _, k := range rv.MapKeys() {
			r.SetMapIndex(k, rv.MapIndex(k))
		}

	} else if v.IsSliceOrArray() {
		r = reflect.MakeSlice(rv.Type(), 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			r = reflect.Append(r, rv.Index(i))
		}

	} else if rv.Kind() == reflect.Struct {
		r = reflect.New(rv.Type())
		rmap := r
		if rmap.Kind() == reflect.Ptr {
			rmap = rmap.Elem()
		}
		for i := 0; i < rv.NumField(); i++ {
			fv := rv.Field(i)
			rmap.Field(i).Set(fv)
		}

		if !wasPtr {
			r = rmap
		}

	} else {
		r = reflect.New(rv.Type())
		rd := r.Elem()
		rd.Set(rv)
		r = rd
	}

	return &Value{val: r, safe: v.safe}
}

// Interface gives you access to the underlying value.
func (v *Value) Interface() any {
	if v.val.IsValid() {
		return v.val.Interface()
	}
	return nil
}

// EqualValueTo checks whether two values are containing the same value or object (if comparable).
func (v *Value) EqualValueTo(other *Value) bool {
	// Handle numeric comparison: float vs int should compare by value (e.g., 8.0 == 8)
	// Also handles uint vs int comparison (see issue #64)
	if v.IsNumber() && other.IsNumber() {
		// If either is a float, compare as floats
		if v.IsFloat() || other.IsFloat() {
			return v.Float() == other.Float()
		}
		// Both are integers (includes uint vs int)
		return v.Integer() == other.Integer()
	}
	if v.IsTime() && other.IsTime() {
		return v.Time().Equal(other.Time())
	}
	// Handle nil/undefined values (see issue #341)
	// Two nil values are considered equal
	if !v.val.IsValid() && !other.val.IsValid() {
		return true
	}
	// One nil and one non-nil are not equal
	if !v.val.IsValid() || !other.val.IsValid() {
		return false
	}
	// Note: reflect.Value.Equal() and Value.Comparable() (Go 1.20+) were considered
	// but benchmarking showed they are slower. Type().Comparable() and
	// Interface() == Interface() is faster due to Go's interface comparison optimization.
	return v.val.CanInterface() && other.val.CanInterface() &&
		v.val.Type().Comparable() && other.val.Type().Comparable() &&
		v.Interface() == other.Interface()
}

// Less implements sort.Interface, and (when the underlying value is a slice) indicates whether the value at index i
// is less than the value at index j.
func (v *Value) Less(i, j int) bool {
	if !v.IsSliceOrArray() || i >= v.Len() || j >= v.Len() {
		return false
	}

	return v.Index(i).Compare(v.Index(j)) == -1
}

// Swap implements sort.Interface, and (when the underlying value is a slice) swaps the elements with indexes i and j.
func (v *Value) Swap(i, j int) {
	if !v.IsSliceOrArray() || i >= v.Len() || j >= v.Len() {
		return
	}

	vi := v.Index(i)
	vj := v.Index(j)
	v.SetIndex(i, vj)
	v.SetIndex(j, vi)
}

func (v *Value) CompareCaseFold(other *Value) int {
	return v.compare(other, false)
}

func (v *Value) Compare(other *Value) int {
	return v.compare(other, true)
}

func (v *Value) compare(other *Value, caseSensitive bool) int {
	if !v.val.IsValid() || !v.val.Type().Comparable() {
		return -1
	}
	if !other.val.IsValid() || !other.val.Type().Comparable() {
		return 1
	}

	switch {
	case v.IsInteger() && other.IsInteger():
		va := v.Int64()
		vb := other.Int64()
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		} else {
			return 0
		}
	case v.IsFloat() && other.IsFloat():
		va := v.Float()
		vb := other.Float()
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		} else {
			return 0
		}
	case v.IsTime() && other.IsTime():
		va := v.Time()
		vb := other.Time()
		if va.Before(vb) {
			return -1
		} else if va.After(vb) {
			return 1
		} else {
			return 0
		}
	case v.IsBool() && other.IsBool():
		va := v.Bool()
		vb := other.Bool()
		if va && !vb {
			return 1
		} else if vb && !va {
			return -1
		} else {
			return 0
		}
	case v.IsSliceOrArray() && other.IsSliceOrArray(), v.IsMap() && other.IsMap():
		va := v.getResolvedValue().Len()
		vb := other.getResolvedValue().Len()
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		} else {
			return 0
		}
	default:
		if v.IsNil() && !other.IsNil() {
			return -1
		} else if other.IsNil() && !v.IsNil() {
			return 1
		}

		va := v.String()
		vb := other.String()

		if !caseSensitive {
			va = strings.ToLower(va)
			vb = strings.ToLower(vb)
		}

		if va < vb {
			return -1
		} else if va > vb {
			return 1
		} else {
			return 0
		}
	}
}

type sortedKeys []reflect.Value

func (sk sortedKeys) Len() int {
	return len(sk)
}

func (sk sortedKeys) Less(i, j int) bool {
	vi := &Value{val: sk[i]}
	vj := &Value{val: sk[j]}
	switch {
	case vi.IsInteger() && vj.IsInteger():
		return vi.Integer() < vj.Integer()
	case vi.IsFloat() && vj.IsFloat():
		return vi.Float() < vj.Float()
	case vi.IsString():
		return vi.String() < vj.String()
	default:
		return vi.Compare(vj) == -1
	}
}

func (sk sortedKeys) Swap(i, j int) {
	sk[i], sk[j] = sk[j], sk[i]
}

type valuesList []*Value

func (vl valuesList) Len() int {
	return len(vl)
}

func (vl valuesList) Less(i, j int) bool {
	vi := vl[i]
	vj := vl[j]
	switch {
	case vi.IsInteger() && vj.IsInteger():
		return vi.Integer() < vj.Integer()
	case vi.IsFloat() && vj.IsFloat():
		return vi.Float() < vj.Float()
	case vi.IsString():
		return vi.String() < vj.String()
	default:
		return vi.Compare(vj) == -1
	}
}

func (vl valuesList) Swap(i, j int) {
	vl[i], vl[j] = vl[j], vl[i]
}
