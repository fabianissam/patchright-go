package playwright

import "testing"

func TestParseEvaluateArgsDefaultsIsolated(t *testing.T) {
	arg, isolated := parseEvaluateArgs([]any{42})
	if arg != 42 {
		t.Fatalf("arg = %v, want 42", arg)
	}
	if !isolated {
		t.Fatal("isolatedContext should default to true")
	}
}

func TestParseEvaluateArgsOptOut(t *testing.T) {
	arg, isolated := parseEvaluateArgs([]any{"hello", EvaluateOptions{IsolatedContext: Bool(false)}})
	if arg != "hello" {
		t.Fatalf("arg = %v, want hello", arg)
	}
	if isolated {
		t.Fatal("isolatedContext should be false when opted out")
	}
}

func TestAddSourceURLToScriptDoesNotLeakPath(t *testing.T) {
	got := addSourceURLToScript("console.log(1)", "/secret/path.js")
	if got != "console.log(1)" {
		t.Fatalf("sourceURL leak: %q", got)
	}
}
