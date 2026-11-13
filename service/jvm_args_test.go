package service

import (
	"reflect"
	"testing"
)

func TestSplitJVMArgs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "", want: []string{}},
		{name: "whitespace only", in: "   \t  ", want: []string{}},
		{name: "simple", in: "-XX:+UseG1GC -Xss1M", want: []string{"-XX:+UseG1GC", "-Xss1M"}},
		{
			name: "quotes stay in token",
			in:   `-Dmsg="hello world" -XX:Foo`,
			want: []string{`-Dmsg="hello`, `world"`, `-XX:Foo`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitJVMArgs(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SplitJVMArgs(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}
