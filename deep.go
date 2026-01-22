package pongo2

import (
	"fmt"
	"reflect"
	"strings"
)

type DeepResolver struct {
	vr  *variableResolver
	ctx *ExecutionContext
}

func (ts *TemplateSet) DeepResolver(ctx Context) *DeepResolver {
	if ctx == nil {
		ctx = Context{}
	}
	vr := &variableResolver{}

	tpl := Template{
		set:         ts,
		isTplString: true,
		Options:     &Options{},
	}
	tpl.Options.Update(ts.Options)
	tpl.Options.DeepResolve = true
	ec := newExecutionContext(&tpl, ctx)

	return &DeepResolver{
		vr:  vr,
		ctx: ec,
	}
}

func (dr *DeepResolver) UpdateContext(ctx Context) {
	dr.ctx.Public.Update(ctx)
}

func (dr *DeepResolver) UpdateOptions(other *Options) {
	dr.ctx.template.Options.Update(other)
}

func (dr *DeepResolver) Evaluate(s string) (*Value, error) {
	return dr.Resolve("{{ " + s + " }}")
}

func (dr *DeepResolver) Resolve(i interface{}) (*Value, error) {
	r, modified, err := dr.vr.resolveInterface(dr.ctx, i)
	if err != nil {
		return nil, err
	}
	if !modified {
		r = i
	}
	return AsValue(r), nil
}

func (vr *variableResolver) stackGet(ctx *ExecutionContext) string {
	var (
		stack []string
		ok    bool
	)
	si := ctx.Private["_resolve_stack"]
	if stack, ok = si.([]string); ok {
		b := strings.Builder{}
		for _, s := range stack {
			if len(s) > 32 {
				s = s[0:32] + " ..."
			}
			b.WriteString(s)
			b.WriteString(" / ")
		}
		return b.String()
	}

	return ""
}
func (vr *variableResolver) stackPush(ctx *ExecutionContext, v interface{}) {
	var (
		stack []string
		ok    bool
	)
	si := ctx.Private["_resolve_stack"]
	if stack, ok = si.([]string); !ok {
		stack = make([]string, 0, 8)
	}
	ctx.Private["_resolve_stack"] = append(stack, fmt.Sprintf("%v", v))
}

func (vr *variableResolver) stackPop(ctx *ExecutionContext) {
	var (
		stack []string
		ok    bool
	)
	si := ctx.Private["_resolve_stack"]
	if stack, ok = si.([]string); !ok {
		return
	}
	ctx.Private["_resolve_stack"] = stack[0 : len(stack)-1]
}

func (vr *variableResolver) resolveDeep(ctx *ExecutionContext, current reflect.Value) (reflect.Value, bool, error) {
	resolved, modified, err := vr.resolveInterface(ctx, current.Interface())
	if err != nil {
		return reflect.Value{}, false, err
	}
	return reflect.ValueOf(resolved), modified, nil
}

func (vr *variableResolver) resolveMap(ctx *ExecutionContext, m map[string]interface{}) (map[string]interface{}, bool, error) {
	vr.stackPush(ctx, m)
	defer func() {
		vr.stackPop(ctx)
	}()

	modified := false

	r := make(map[string]interface{})
	for k, v := range m {
		newV, elementModified, err := vr.resolveInterface(ctx, v)
		if err != nil {
			return nil, false, err
		}
		modified = modified || elementModified
		r[k] = newV
	}
	return r, modified, nil
}

func (vr *variableResolver) resolveSlice(ctx *ExecutionContext, s []interface{}) ([]interface{}, bool, error) {
	vr.stackPush(ctx, s)
	defer func() {
		vr.stackPop(ctx)
	}()

	modified := false

	r := make([]interface{}, len(s))
	for k, v := range s {
		newV, elementModified, err := vr.resolveInterface(ctx, v)
		if err != nil {
			return nil, false, err
		}
		modified = modified || elementModified
		r[k] = newV
	}
	return r, modified, nil
}

func (vr *variableResolver) resolveInterface(ctx *ExecutionContext, i interface{}) (interface{}, bool, error) {
	vr.stackPush(ctx, i)
	defer func() {
		vr.stackPop(ctx)
	}()

	switch it := i.(type) {
	case map[string]interface{}:
		return vr.resolveMap(ctx, it)

	case Context:
		return vr.resolveMap(ctx, it)

	case []interface{}:
		return vr.resolveSlice(ctx, it)

	case MultiPart:
		resolved, _, err := vr.resolveSlice(ctx, it)
		if err != nil {
			return nil, false, err
		}

		b := strings.Builder{}
		for _, v := range resolved {
			val := AsValue(v)
			b.WriteString(val.String())
		}

		return b.String(), true, nil

	case *Template:
		it.Options.Update(&Options{
			DeepResolve:      ctx.DeepResolve,
			AutoescapeFilter: ctx.template.Options.AutoescapeFilter,
		})

		resolved, err := it.Evaluate(ctx.Public)
		if err != nil {
			return nil, false, err
		}

		return resolved, true, err

	case string:
		if !strings.Contains(it, "{{") && !strings.Contains(it, "{%") {
			return it, false, nil
		}

		tpl, err := ctx.template.set.FromString(it)
		if err != nil {
			return nil, false, err
		}
		tpl.Options.Update(&Options{
			DeepResolve:      ctx.DeepResolve,
			AutoescapeFilter: ctx.template.Options.AutoescapeFilter,
		})
		resolved, err := tpl.Evaluate(ctx.Public)
		if err != nil {
			return nil, false, err
		}
		if s, ok := resolved.(string); ok && s == it {
			return resolved, false, nil
		}

		r, _, err := vr.resolveInterface(ctx, resolved)

		if s, ok := r.(string); ok && s == it {
			return r, false, nil
		}

		return r, true, err

	default:
		v := AsValue(i)
		if v.IsMap() || v.IsSliceOrArray() {
			modified := false
			// custom map/slice type (not map[string]interface{}/[]interface{}
			newVal, err := v.Map(func(idx, count int, key, value *Value) (newKey, newValue *Value, err error) {
				val, elementModified, e := vr.resolveInterface(ctx, value.Interface())
				if e != nil {
					return nil, nil, &Error{OrigError: e}
				}
				modified = modified || elementModified
				return key, AsValue(val), nil
			})
			if err != nil {
				return nil, false, err
			}
			return newVal.Interface(), modified, nil
		} else {
			return i, false, nil
		}
	}
}
