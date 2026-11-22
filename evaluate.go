package playwright

// EvaluateOptions is a Patchright extension of the Playwright evaluate APIs.
// Pass it as an extra argument after the JavaScript argument to control whether
// the expression runs in an isolated execution context. Patchright defaults to
// isolatedContext=true so pages cannot observe Runtime.enable.
type EvaluateOptions struct {
	IsolatedContext *bool
}

func parseEvaluateArgs(options []any) (arg any, isolatedContext bool) {
	isolatedContext = true
	for _, o := range options {
		switch v := o.(type) {
		case EvaluateOptions:
			if v.IsolatedContext != nil {
				isolatedContext = *v.IsolatedContext
			}
		case *EvaluateOptions:
			if v != nil && v.IsolatedContext != nil {
				isolatedContext = *v.IsolatedContext
			}
		default:
			arg = o
		}
	}
	return arg, isolatedContext
}

func evaluateExpressionParams(expression string, options []any) map[string]any {
	arg, isolated := parseEvaluateArgs(options)
	return map[string]any{
		"expression":      expression,
		"arg":             serializeArgument(arg),
		"isolatedContext": isolated,
	}
}

func isolatedContextOrDefault(value *bool) bool {
	if value != nil {
		return *value
	}
	return true
}
