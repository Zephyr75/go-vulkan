package vk

/*
#include <vulkan/vulkan.h>
*/
import "C"

import "unsafe"

// CreateFence creates a VkFence. Pass FenceCreateSignaled to start signaled
// (the frames-in-flight idiom waits on the fence before recording frame 0).
func CreateFence(d Device, flags FenceCreateFlags) (Fence, error) {
	info := C.VkFenceCreateInfo{
		sType: C.VK_STRUCTURE_TYPE_FENCE_CREATE_INFO,
		flags: C.VkFenceCreateFlags(flags),
	}
	var out C.VkFence
	if err := check(C.vkCreateFence(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return Fence(unsafe.Pointer(out)), nil
}

func DestroyFence(d Device, f Fence) {
	C.vkDestroyFence(C.VkDevice(unsafe.Pointer(d)), C.VkFence(unsafe.Pointer(f)), nil)
}

func CreateSemaphore(d Device) (Semaphore, error) {
	info := C.VkSemaphoreCreateInfo{sType: C.VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO}
	var out C.VkSemaphore
	if err := check(C.vkCreateSemaphore(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return Semaphore(unsafe.Pointer(out)), nil
}

func DestroySemaphore(d Device, s Semaphore) {
	C.vkDestroySemaphore(C.VkDevice(unsafe.Pointer(d)), C.VkSemaphore(unsafe.Pointer(s)), nil)
}

// WaitForFences blocks until all (waitAll) or any of the fences signal, or the
// timeout (ns) elapses. A timeout returns vk.Timeout as the error.
func WaitForFences(d Device, fences []Fence, waitAll bool, timeout uint64) error {
	if len(fences) == 0 {
		return nil
	}
	all := C.VkBool32(C.VK_FALSE)
	if waitAll {
		all = C.VK_TRUE
	}
	return check(C.vkWaitForFences(C.VkDevice(unsafe.Pointer(d)),
		C.uint32_t(len(fences)),
		(*C.VkFence)(unsafe.Pointer(&fences[0])),
		all, C.uint64_t(timeout)))
}

func ResetFences(d Device, fences []Fence) error {
	if len(fences) == 0 {
		return nil
	}
	return check(C.vkResetFences(C.VkDevice(unsafe.Pointer(d)),
		C.uint32_t(len(fences)),
		(*C.VkFence)(unsafe.Pointer(&fences[0]))))
}
