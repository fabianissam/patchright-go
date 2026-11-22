package main

import "testing"

func TestBumpPlaywrightCliVersion(t *testing.T) {
	in := "const (\n\tplaywrightCliVersion = \"1.64.3\"\n)\n"
	got, err := bumpPlaywrightCliVersion(in, "1.65.0")
	if err != nil {
		t.Fatal(err)
	}
	want := "const (\n\tplaywrightCliVersion = \"1.65.0\"\n)\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCmpVersion(t *testing.T) {
	if cmpVersion("1.64.4", "1.64.3") <= 0 {
		t.Fatal("expected 1.64.4 > 1.64.3")
	}
	if cmpVersion("1.64.3", "1.64.3") != 0 {
		t.Fatal("expected equal")
	}
	if cmpVersion("1.63.0", "1.64.3") >= 0 {
		t.Fatal("expected older")
	}
}

func TestCurrentCLIVersion(t *testing.T) {
	got, err := currentCLIVersion("playwrightCliVersion = \"1.64.3\"")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.64.3" {
		t.Fatalf("got %s", got)
	}
}
