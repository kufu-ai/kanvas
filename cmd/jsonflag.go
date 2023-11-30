package cmd

// Shamelessly copied from https://stackoverflow.com/a/70542226.
// Thanks to https://stackoverflow.com/users/3163818/applejag for the answer!

import (
	"encoding/json"
)

// JSONFlag is a flag that accepts JSON
// You can use it like this:
//
//	type MyObject struct {
//		Foo string `json:"foo"`
//	}
//	var myObject MyObject
//	cmd.Flags().Var(&JSONFlag{&myObject}, "my-flag", "my flag")
type JSONFlag struct {
	Target interface{}
}

// String is used both by fmt.Print and by Cobra in help text
func (f *JSONFlag) String() string {
	b, err := json.Marshal(f.Target)
	if err != nil {
		return "failed to marshal object"
	}
	return string(b)
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (f *JSONFlag) Set(v string) error {
	return json.Unmarshal([]byte(v), f.Target)
}

// Type is only used in help text
func (f *JSONFlag) Type() string {
	return "json"
}
