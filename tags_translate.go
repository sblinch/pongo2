package pongo2

import (
	"fmt"
)

type TranslateFunc = func(string, ...any) string

type tagTranslateNode struct {
	as   string
	msg  IEvaluator
	args []IEvaluator
}

func (node *tagTranslateNode) Execute(ctx *ExecutionContext, writer TemplateWriter) error {
	value, err := node.msg.Evaluate(ctx)
	if err != nil {
		return err
	}

	var args []interface{}
	if len(node.args) > 0 {
		args = make([]interface{}, 0, len(node.args))
		for _, arg := range node.args {
			value, err := arg.Evaluate(ctx)
			if err != nil {
				return err
			}
			args = append(args, value.Interface())
		}
	}

	f := ctx.Translator
	if f == nil {
		f = fmt.Sprintf
	}

	msg := f(value.String(), args...)

	if node.as != "" {
		ctx.Private[node.as] = msg
	} else {
		_, _ = writer.WriteString(msg)
	}
	return nil
}

func tagTranslateParser(doc *Parser, start *Token, arguments *Parser) (INodeTag, error) {
	node := &tagTranslateNode{}

	// Variable expression
	msg, err := arguments.ParseExpression()
	if err != nil {
		return nil, err
	}
	node.msg = msg

	for {
		if arguments.Match(TokenSymbol, ",") == nil {
			break
		}
		if node.args == nil {
			node.args = make([]IEvaluator, 0, 8)
		}

		v, err := arguments.ParseExpression()
		if err != nil {
			return nil, err
		}
		node.args = append(node.args, v)
	}

	if arguments.Match(TokenKeyword, "as") != nil {
		// Parse alias
		asNameToken := arguments.MatchType(TokenIdentifier)
		if asNameToken == nil {
			return nil, arguments.Error("Expected an identifier.", nil)
		}
		node.as = asNameToken.Val
	}

	//
	// // Parse variable name
	// typeToken := arguments.MatchType(TokenIdentifier)
	// if typeToken == nil {
	// 	return nil, arguments.Error("Expected an identifier.", nil)
	// }
	// node.name = typeToken.Val
	//
	// if arguments.Match(TokenSymbol, "=") == nil {
	// 	return nil, arguments.Error("Expected '='.", nil)
	// }

	// Remaining arguments
	if arguments.Remaining() > 0 {
		return nil, arguments.Error("Malformed 'translate'-tag arguments.", nil)
	}

	return node, nil
}

func init() {
	mustRegisterTag("translate", tagTranslateParser)
}
