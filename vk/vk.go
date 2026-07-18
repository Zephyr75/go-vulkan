// Package vk is a hand-written cgo binding for the subset of Vulkan 1.3 used by
// the "How to Vulkan 2026" tutorial: dynamic rendering, buffer device address,
// descriptor indexing and synchronization2
package vk

/*
#cgo LDFLAGS: -lvulkan
#include <stdlib.h>
#include <string.h>
#include <vulkan/vulkan.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// Wraps VkResult and implements error for the non-success codes
type Result int32

// Result codes the app must branch on
const (
	Success                 = Result(C.VK_SUCCESS)
	NotReady                = Result(C.VK_NOT_READY)
	Timeout                 = Result(C.VK_TIMEOUT)
	ErrOutOfDateKHR         = Result(C.VK_ERROR_OUT_OF_DATE_KHR)
	SuboptimalKHR           = Result(C.VK_SUBOPTIMAL_KHR)
	ErrDeviceLost           = Result(C.VK_ERROR_DEVICE_LOST)
	ErrOutOfHostMemory      = Result(C.VK_ERROR_OUT_OF_HOST_MEMORY)
	ErrOutOfDeviceMemory    = Result(C.VK_ERROR_OUT_OF_DEVICE_MEMORY)
	ErrInitializationFailed = Result(C.VK_ERROR_INITIALIZATION_FAILED)
)

func (r Result) Error() string { return r.String() }

func (r Result) String() string {
	switch r {
	case Success:
		return "VK_SUCCESS"
	case NotReady:
		return "VK_NOT_READY"
	case Timeout:
		return "VK_TIMEOUT"
	case ErrOutOfDateKHR:
		return "VK_ERROR_OUT_OF_DATE_KHR"
	case SuboptimalKHR:
		return "VK_SUBOPTIMAL_KHR"
	case ErrDeviceLost:
		return "VK_ERROR_DEVICE_LOST"
	case ErrOutOfHostMemory:
		return "VK_ERROR_OUT_OF_HOST_MEMORY"
	case ErrOutOfDeviceMemory:
		return "VK_ERROR_OUT_OF_DEVICE_MEMORY"
	case ErrInitializationFailed:
		return "VK_ERROR_INITIALIZATION_FAILED"
	default:
		return fmt.Sprintf("VkResult(%d)", int32(r))
	}
}

// Turns a VkResult into a Go error with nil on success
func check(r C.VkResult) error {
	if r == C.VK_SUCCESS {
		return nil
	}
	return Result(int32(r))
}

// ---- handle types --------------------------------------------------------
//
// On LP64 every Vulkan handle (dispatchable or not) is a pointer, so a uintptr
// alias represents them all and lets external code (GLFW's surface creation)
// construct them. Conversion to the cgo pointer types goes through
// unsafe.Pointer at the call boundary.

type (
	Instance            uintptr
	PhysicalDevice      uintptr
	Device              uintptr
	Queue               uintptr
	SurfaceKHR          uintptr
	SwapchainKHR        uintptr
	Image               uintptr
	ImageView           uintptr
	DeviceMemory        uintptr
	Buffer              uintptr
	Sampler             uintptr
	Fence               uintptr
	Semaphore           uintptr
	CommandPool         uintptr
	CommandBuffer       uintptr
	DescriptorSetLayout uintptr
	DescriptorPool      uintptr
	DescriptorSet       uintptr
	PipelineLayout      uintptr
	Pipeline            uintptr
	ShaderModule        uintptr
)

// ---- string array marshaling --------------------------------------------

// Converts a []string to a C **char plus count; call the returned free function (deferred) after the C call that consumes the array
func cStrings(ss []string) (**C.char, C.uint32_t, func()) {
	if len(ss) == 0 {
		return nil, 0, func() {}
	}
	arr := C.malloc(C.size_t(len(ss)) * C.size_t(unsafe.Sizeof(uintptr(0))))
	slice := unsafe.Slice((**C.char)(arr), len(ss))
	for i, s := range ss {
		slice[i] = C.CString(s)
	}
	free := func() {
		for i := range ss {
			C.free(unsafe.Pointer(slice[i]))
		}
		C.free(arr)
	}
	return (**C.char)(arr), C.uint32_t(len(ss)), free
}

// ---- two-call enumeration helper ----------------------------------------

// Runs the classic Vulkan (count, nil) then (count, ptr) two-call pattern and returns the filled slice
func enumerate[T any](f func(count *C.uint32_t, out *T) C.VkResult) ([]T, error) {
	var count C.uint32_t
	if err := check(f(&count, nil)); err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	out := make([]T, count)
	if err := check(f(&count, &out[0])); err != nil {
		return nil, err
	}
	return out[:count], nil
}

// Runs the two-call pattern for enumeration functions that return no VkResult
func enumerateVoid[T any](f func(count *C.uint32_t, out *T)) []T {
	var count C.uint32_t
	f(&count, nil)
	if count == 0 {
		return nil
	}
	out := make([]T, count)
	f(&count, &out[0])
	return out[:count]
}

// Copies a Go slice into a mapped device pointer
func MemCopy[T any](dst unsafe.Pointer, src []T) {
	if len(src) == 0 {
		return
	}
	n := uintptr(len(src)) * unsafe.Sizeof(src[0])
	C.memcpy(dst, unsafe.Pointer(&src[0]), C.size_t(n))
}
