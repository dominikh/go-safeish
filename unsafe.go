// Package safeish provides safe-ish unsafe helpers.
package safeish

import (
	"fmt"
	"strings"
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

func FindNull(s *byte) int {
	if s == nil {
		return 0
	}

	// pageSize is the unit we scan at a time looking for NULL.
	// It must be the minimum page size for any architecture Go
	// runs on. It's okay (just a minor performance loss) if the
	// actual system page size is larger than this value.
	const pageSize = 4096

	offset := 0
	ptr := unsafe.Pointer(s)
	// IndexByteString uses wide reads, so we need to be careful
	// with page boundaries. Call IndexByteString on
	// [ptr, endOfPage) interval.
	safeLen := int(pageSize - uintptr(ptr)%pageSize)
	for {
		t := unsafe.String((*byte)(ptr), safeLen)
		// Check one page at a time.
		if i := strings.IndexByte(t, 0); i != -1 {
			return offset + i
		}
		// Move to next page
		ptr = unsafe.Pointer(uintptr(ptr) + uintptr(safeLen))
		offset += safeLen
		safeLen = pageSize
	}
}

// SliceCastPtr casts a slice of underlying type []SrcE to a pointer of
// underlying type *DstE to the slice's first element, or nil if the slice's
// capacity is 0. It ensures that the pointer doesn't extend past the end of the
// slice.
func SliceCastPtr[Dst ~*DstE, Src ~[]SrcE, DstE, SrcE any](x Src) Dst {
	if cap(x) == 0 {
		return nil
	}
	type sliceHeader struct {
		data unsafe.Pointer
		len  int
		cap  int
	}

	sizeSrc := unsafe.Sizeof(*new(SrcE))
	sizeDst := unsafe.Sizeof(*new(DstE))

	if sizeSrc != sizeDst {
		// This check gets eliminated by the compiler when the sizes match, but
		// the inliner doesn't know that. GOEXPERIMENT=newinliner claims that
		// this function is inlinable, but it doesn't actually get inlined.

		if sz := int(sizeSrc) * cap(x); sz < int(sizeDst) {
			panic(
				fmt.Sprintf("slice has capacity of %d bytes, but a single %T is %d bytes",
					sz, *new(DstE), sizeDst))
		}
	}

	// This way of getting the pointer has lower inlining complexity than
	// &x[:1][0]
	ptrDst := (*sliceHeader)(unsafe.Pointer(&x)).data
	return Dst(ptrDst)
}
