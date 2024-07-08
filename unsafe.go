// Package safeish provides safe-ish unsafe helpers.
package safeish

import (
	"unsafe"

	"golang.org/x/exp/constraints"
)

// Cast casts x from type Src to type Dst. It uses generics to provide a
// syntactic alternative to the common unsafe.Pointer conversion pattern.
//
// Example:
//
//	var x Foo
//
//	_ = Cast[Bar](x)
//	// the above is identical to the below
//	_ = *(*Bar)(unsafe.Pointer(&x))
func Cast[Dst, Src any](x Src) Dst {
	return *(*Dst)(unsafe.Pointer(&x))
}

// SliceCast casts a slice of underlying type []SrcE to a slice of underlying
// type []DstE, automatically adjusting the length and capacity based on the
// ratio of sizeof(SrcE) to sizeof(DstE). sizeof(DstE) may be both larger or
// smaller than sizeof(SrcE).
//
// The ratio is expected to be integer, but non-integer ratios will not cause
// invalid memory accesses.
//
// The type parameters are ordered so that at most the first one has to be
// provided explicitly.
//
// SliceCast is fully inlinable.
func SliceCast[Dst ~[]DstE, Src ~[]SrcE, DstE, SrcE any](x Src) Dst {
	// We don't use our Cast helper in this function because it increases the
	// function complexity, making inlining more difficult.

	type sliceHeader struct {
		data unsafe.Pointer
		len  int
		cap  int
	}

	if cap(x) == 0 {
		return nil
	}

	// This way of getting the pointer has lower inlining complexity than
	// &x[:1][0]
	ptrDst := (*sliceHeader)(unsafe.Pointer(&x)).data

	sizeSrc := unsafe.Sizeof(*new(SrcE))
	sizeDst := unsafe.Sizeof(*new(DstE))

	if sizeSrc >= sizeDst {
		return *(*Dst)(unsafe.Pointer(&sliceHeader{
			data: ptrDst,
			len:  len(x) * int(sizeSrc/sizeDst),
			cap:  cap(x) * int(sizeSrc/sizeDst),
		}))
	} else {
		return *(*Dst)(unsafe.Pointer(&sliceHeader{
			data: ptrDst,
			len:  len(x) / int(sizeDst/sizeSrc),
			cap:  cap(x) / int(sizeDst/sizeSrc),
		}))
	}
}

// Index provides unsafe slice indexing without bounds checks. This function has
// absolutely no safety checks.
func Index[E any, S ~[]E, Int constraints.Integer](ptr S, idx Int) *E {
	offset := unsafe.Sizeof(*new(E)) * uintptr(idx)
	return (*E)(unsafe.Add(unsafe.Pointer(&ptr[0]), offset))
}

// AsBytes returns the underlying byte representation of the value pointed to by
// ptr.
func AsBytes[E any, T *E](ptr T) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), unsafe.Sizeof(*ptr))
}
