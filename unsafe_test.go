package safeish

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func ExampleSliceCast() {
	type S struct {
		A, B uint32
	}

	x := make([]byte, 32, 60)
	s := SliceCast[[]S](x)
	fmt.Println(len(s), cap(s))
	// Output:
	// 4 7
}

func TestAsBytes(t *testing.T) {
	type X struct {
		A uint32
		B uint64
		C uint32
	}

	var x = X{1, 2, 3}
	b := AsBytes(&x)
	want := []byte{
		1, 0, 0, 0, // A
		0, 0, 0, 0, // padding
		2, 0, 0, 0, 0, 0, 0, 0, // B
		3, 0, 0, 0, // C
		0, 0, 0, 0, // padding
	}
	if diff := cmp.Diff(want, b); diff != "" {
		t.Error(diff)
	}
}
