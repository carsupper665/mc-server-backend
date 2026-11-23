package common

import (
	"reflect"
	"testing"
)

func TestSplitList(t *testing.T) {
	got := SplitList(" 127.0.0.1, ,10.0.0.0/8,")
	want := []string{"127.0.0.1", "10.0.0.0/8"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SplitList = %v, want %v", got, want)
	}
	if SplitList("") != nil {
		t.Fatal("empty value should give nil (trust no proxies)")
	}
}
