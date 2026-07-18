package vk

/*
#include <stdlib.h>
#include <vulkan/vulkan.h>
*/
import "C"

import "unsafe"

func CreateCommandPool(d Device, queueFamily uint32, flags CommandPoolCreateFlags) (CommandPool, error) {
	info := C.VkCommandPoolCreateInfo{
		sType:            C.VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
		flags:            C.VkCommandPoolCreateFlags(flags),
		queueFamilyIndex: C.uint32_t(queueFamily),
	}
	var out C.VkCommandPool
	if err := check(C.vkCreateCommandPool(C.VkDevice(unsafe.Pointer(d)), &info, nil, &out)); err != nil {
		return 0, err
	}
	return CommandPool(unsafe.Pointer(out)), nil
}

func DestroyCommandPool(d Device, p CommandPool) {
	C.vkDestroyCommandPool(C.VkDevice(unsafe.Pointer(d)), C.VkCommandPool(unsafe.Pointer(p)), nil)
}

// Allocates count primary command buffers from the pool
func AllocateCommandBuffers(d Device, pool CommandPool, count uint32) ([]CommandBuffer, error) {
	info := C.VkCommandBufferAllocateInfo{
		sType:              C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		commandPool:        C.VkCommandPool(unsafe.Pointer(pool)),
		level:              C.VK_COMMAND_BUFFER_LEVEL_PRIMARY,
		commandBufferCount: C.uint32_t(count),
	}
	out := make([]CommandBuffer, count)
	if count == 0 {
		return out, nil
	}
	if err := check(C.vkAllocateCommandBuffers(C.VkDevice(unsafe.Pointer(d)), &info,
		(*C.VkCommandBuffer)(unsafe.Pointer(&out[0])))); err != nil {
		return nil, err
	}
	return out, nil
}

func ResetCommandBuffer(cb CommandBuffer) error {
	return check(C.vkResetCommandBuffer(C.VkCommandBuffer(unsafe.Pointer(cb)), 0))
}

func BeginCommandBuffer(cb CommandBuffer, flags CommandBufferUsageFlags) error {
	info := C.VkCommandBufferBeginInfo{
		sType: C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
		flags: C.VkCommandBufferUsageFlags(flags),
	}
	return check(C.vkBeginCommandBuffer(C.VkCommandBuffer(unsafe.Pointer(cb)), &info))
}

func EndCommandBuffer(cb CommandBuffer) error {
	return check(C.vkEndCommandBuffer(C.VkCommandBuffer(unsafe.Pointer(cb))))
}

// ---- synchronization2 submit ---------------------------------------------

// SemaphoreSubmitInfo names a semaphore to wait on / signal, with the pipeline
// stage the wait/signal is scoped to. Value is for timeline semaphores (0 for
// binary).
type SemaphoreSubmitInfo struct {
	Semaphore Semaphore
	Value     uint64
	StageMask PipelineStageFlags2
}

// SubmitInfo2 is one sync2 batch: wait on semaphores, run command buffers,
// signal semaphores.
type SubmitInfo2 struct {
	WaitSemaphores   []SemaphoreSubmitInfo
	CommandBuffers   []CommandBuffer
	SignalSemaphores []SemaphoreSubmitInfo
}

// Marshals a semaphore submit-info array into the arena
func semaphoreInfosC(a *arena, infos []SemaphoreSubmitInfo) *C.VkSemaphoreSubmitInfo {
	n := len(infos)
	if n == 0 {
		return nil
	}
	p := (*C.VkSemaphoreSubmitInfo)(a.alloc(n, unsafe.Sizeof(C.VkSemaphoreSubmitInfo{})))
	s := unsafe.Slice(p, n)
	for i, in := range infos {
		s[i].sType = C.VK_STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO
		s[i].semaphore = C.VkSemaphore(unsafe.Pointer(in.Semaphore))
		s[i].value = C.uint64_t(in.Value)
		s[i].stageMask = C.VkPipelineStageFlags2(in.StageMask)
	}
	return p
}

// Marshals a command-buffer submit-info array into the arena
func commandBufferInfosC(a *arena, cbs []CommandBuffer) *C.VkCommandBufferSubmitInfo {
	n := len(cbs)
	if n == 0 {
		return nil
	}
	p := (*C.VkCommandBufferSubmitInfo)(a.alloc(n, unsafe.Sizeof(C.VkCommandBufferSubmitInfo{})))
	s := unsafe.Slice(p, n)
	for i, cb := range cbs {
		s[i].sType = C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_SUBMIT_INFO
		s[i].commandBuffer = C.VkCommandBuffer(unsafe.Pointer(cb))
	}
	return p
}

// Submits sync2 batches, optionally signaling fence on completion
func QueueSubmit2(q Queue, submits []SubmitInfo2, fence Fence) error {
	var a arena
	defer a.free()

	n := len(submits)
	sub := (*C.VkSubmitInfo2)(a.alloc(n, unsafe.Sizeof(C.VkSubmitInfo2{})))
	ss := unsafe.Slice(sub, n)
	for i, s := range submits {
		ss[i].sType = C.VK_STRUCTURE_TYPE_SUBMIT_INFO_2
		ss[i].waitSemaphoreInfoCount = C.uint32_t(len(s.WaitSemaphores))
		ss[i].pWaitSemaphoreInfos = semaphoreInfosC(&a, s.WaitSemaphores)
		ss[i].commandBufferInfoCount = C.uint32_t(len(s.CommandBuffers))
		ss[i].pCommandBufferInfos = commandBufferInfosC(&a, s.CommandBuffers)
		ss[i].signalSemaphoreInfoCount = C.uint32_t(len(s.SignalSemaphores))
		ss[i].pSignalSemaphoreInfos = semaphoreInfosC(&a, s.SignalSemaphores)
	}

	return check(C.vkQueueSubmit2(C.VkQueue(unsafe.Pointer(q)),
		C.uint32_t(n), sub, C.VkFence(unsafe.Pointer(fence))))
}

func QueueWaitIdle(q Queue) error {
	return check(C.vkQueueWaitIdle(C.VkQueue(unsafe.Pointer(q))))
}
