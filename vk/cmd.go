package vk

/*
#include <stdlib.h>
#include <string.h>
#include <vulkan/vulkan.h>
*/
import "C"

import "unsafe"

// ---- clear value union ---------------------------------------------------

// ClearValue is the raw 16-byte VkClearValue union. Build one with
// ClearColor or ClearDepthStencil.
type ClearValue [16]byte

// ClearColor builds a float RGBA color clear value.
func ClearColor(r, g, b, a float32) ClearValue {
	var cv ClearValue
	f := (*[4]float32)(unsafe.Pointer(&cv[0]))
	f[0], f[1], f[2], f[3] = r, g, b, a
	return cv
}

// ClearDepthStencil builds a depth/stencil clear value.
func ClearDepthStencil(depth float32, stencil uint32) ClearValue {
	var cv ClearValue
	*(*float32)(unsafe.Pointer(&cv[0])) = depth
	*(*uint32)(unsafe.Pointer(&cv[4])) = stencil
	return cv
}

func (cv ClearValue) c() C.VkClearValue {
	return *(*C.VkClearValue)(unsafe.Pointer(&cv[0]))
}

// ---- sync2 image barriers ------------------------------------------------

// ImageMemoryBarrier2 is a synchronization2 image barrier. Queue family fields
// default to QueueFamilyIgnored when left zero-valued via the helper below; set
// them explicitly for a queue ownership transfer.
type ImageMemoryBarrier2 struct {
	SrcStageMask        PipelineStageFlags2
	SrcAccessMask       AccessFlags2
	DstStageMask        PipelineStageFlags2
	DstAccessMask       AccessFlags2
	OldLayout           ImageLayout
	NewLayout           ImageLayout
	SrcQueueFamilyIndex uint32
	DstQueueFamilyIndex uint32
	Image               Image
	SubresourceRange    ImageSubresourceRange
}

// CmdPipelineBarrier2 records a VkDependencyInfo carrying image barriers. This
// is the single layout-transition primitive reused across the project.
func CmdPipelineBarrier2(cb CommandBuffer, barriers []ImageMemoryBarrier2) {
	if len(barriers) == 0 {
		return
	}
	arr := (*C.VkImageMemoryBarrier2)(C.calloc(C.size_t(len(barriers)), C.size_t(unsafe.Sizeof(C.VkImageMemoryBarrier2{}))))
	defer C.free(unsafe.Pointer(arr))
	s := unsafe.Slice(arr, len(barriers))
	for i, b := range barriers {
		s[i].sType = C.VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER_2
		s[i].srcStageMask = C.VkPipelineStageFlags2(b.SrcStageMask)
		s[i].srcAccessMask = C.VkAccessFlags2(b.SrcAccessMask)
		s[i].dstStageMask = C.VkPipelineStageFlags2(b.DstStageMask)
		s[i].dstAccessMask = C.VkAccessFlags2(b.DstAccessMask)
		s[i].oldLayout = C.VkImageLayout(b.OldLayout)
		s[i].newLayout = C.VkImageLayout(b.NewLayout)
		s[i].srcQueueFamilyIndex = C.uint32_t(b.SrcQueueFamilyIndex)
		s[i].dstQueueFamilyIndex = C.uint32_t(b.DstQueueFamilyIndex)
		s[i].image = C.VkImage(unsafe.Pointer(b.Image))
		s[i].subresourceRange = C.VkImageSubresourceRange{
			aspectMask:     C.VkImageAspectFlags(b.SubresourceRange.AspectMask),
			baseMipLevel:   C.uint32_t(b.SubresourceRange.BaseMipLevel),
			levelCount:     C.uint32_t(b.SubresourceRange.LevelCount),
			baseArrayLayer: C.uint32_t(b.SubresourceRange.BaseArrayLayer),
			layerCount:     C.uint32_t(b.SubresourceRange.LayerCount),
		}
	}
	dep := C.VkDependencyInfo{
		sType:                   C.VK_STRUCTURE_TYPE_DEPENDENCY_INFO,
		imageMemoryBarrierCount: C.uint32_t(len(barriers)),
		pImageMemoryBarriers:    arr,
	}
	C.vkCmdPipelineBarrier2(C.VkCommandBuffer(unsafe.Pointer(cb)), &dep)
}

// ---- buffer -> image copy ------------------------------------------------

// BufferImageCopy is a single copy region: whole tightly-packed source at
// bufferOffset into the image mip/layer at the given offset/extent.
type BufferImageCopy struct {
	BufferOffset      uint64
	BufferRowLength   uint32
	BufferImageHeight uint32
	AspectMask        ImageAspectFlags
	MipLevel          uint32
	BaseArrayLayer    uint32
	LayerCount        uint32
	ImageOffset       Offset2D
	ImageExtent       Extent3D
}

func CmdCopyBufferToImage(cb CommandBuffer, src Buffer, dst Image, layout ImageLayout, regions []BufferImageCopy) {
	if len(regions) == 0 {
		return
	}
	arr := (*C.VkBufferImageCopy)(C.calloc(C.size_t(len(regions)), C.size_t(unsafe.Sizeof(C.VkBufferImageCopy{}))))
	defer C.free(unsafe.Pointer(arr))
	s := unsafe.Slice(arr, len(regions))
	for i, r := range regions {
		lc := r.LayerCount
		if lc == 0 {
			lc = 1
		}
		s[i].bufferOffset = C.VkDeviceSize(r.BufferOffset)
		s[i].bufferRowLength = C.uint32_t(r.BufferRowLength)
		s[i].bufferImageHeight = C.uint32_t(r.BufferImageHeight)
		s[i].imageSubresource = C.VkImageSubresourceLayers{
			aspectMask:     C.VkImageAspectFlags(r.AspectMask),
			mipLevel:       C.uint32_t(r.MipLevel),
			baseArrayLayer: C.uint32_t(r.BaseArrayLayer),
			layerCount:     C.uint32_t(lc),
		}
		s[i].imageOffset = C.VkOffset3D{x: C.int32_t(r.ImageOffset.X), y: C.int32_t(r.ImageOffset.Y), z: 0}
		s[i].imageExtent = C.VkExtent3D{
			width:  C.uint32_t(r.ImageExtent.Width),
			height: C.uint32_t(r.ImageExtent.Height),
			depth:  C.uint32_t(r.ImageExtent.Depth),
		}
	}
	C.vkCmdCopyBufferToImage(C.VkCommandBuffer(unsafe.Pointer(cb)),
		C.VkBuffer(unsafe.Pointer(src)), C.VkImage(unsafe.Pointer(dst)),
		C.VkImageLayout(layout), C.uint32_t(len(regions)), arr)
}

// ---- dynamic rendering ---------------------------------------------------

// RenderingAttachmentInfo describes one color or depth attachment for
// CmdBeginRendering.
type RenderingAttachmentInfo struct {
	ImageView   ImageView
	ImageLayout ImageLayout
	LoadOp      AttachmentLoadOp
	StoreOp     AttachmentStoreOp
	ClearValue  ClearValue
}

// RenderingInfo is the Go-facing input to CmdBeginRendering. DepthAttachment is
// optional.
type RenderingInfo struct {
	RenderArea       Rect2D
	LayerCount       uint32
	ColorAttachments []RenderingAttachmentInfo
	DepthAttachment  *RenderingAttachmentInfo
}

func fillAttachment(dst *C.VkRenderingAttachmentInfo, a *RenderingAttachmentInfo) {
	dst.sType = C.VK_STRUCTURE_TYPE_RENDERING_ATTACHMENT_INFO
	dst.imageView = C.VkImageView(unsafe.Pointer(a.ImageView))
	dst.imageLayout = C.VkImageLayout(a.ImageLayout)
	dst.loadOp = C.VkAttachmentLoadOp(a.LoadOp)
	dst.storeOp = C.VkAttachmentStoreOp(a.StoreOp)
	dst.clearValue = a.ClearValue.c()
}

func CmdBeginRendering(cb CommandBuffer, ri RenderingInfo) {
	lc := ri.LayerCount
	if lc == 0 {
		lc = 1
	}
	info := C.VkRenderingInfo{
		sType:      C.VK_STRUCTURE_TYPE_RENDERING_INFO,
		layerCount: C.uint32_t(lc),
	}
	info.renderArea = C.VkRect2D{
		offset: C.VkOffset2D{x: C.int32_t(ri.RenderArea.Offset.X), y: C.int32_t(ri.RenderArea.Offset.Y)},
		extent: C.VkExtent2D{width: C.uint32_t(ri.RenderArea.Extent.Width), height: C.uint32_t(ri.RenderArea.Extent.Height)},
	}

	var colorArr *C.VkRenderingAttachmentInfo
	if n := len(ri.ColorAttachments); n > 0 {
		colorArr = (*C.VkRenderingAttachmentInfo)(C.calloc(C.size_t(n), C.size_t(unsafe.Sizeof(C.VkRenderingAttachmentInfo{}))))
		defer C.free(unsafe.Pointer(colorArr))
		s := unsafe.Slice(colorArr, n)
		for i := range ri.ColorAttachments {
			fillAttachment(&s[i], &ri.ColorAttachments[i])
		}
		info.colorAttachmentCount = C.uint32_t(n)
		info.pColorAttachments = colorArr
	}

	var depth *C.VkRenderingAttachmentInfo
	if ri.DepthAttachment != nil {
		depth = (*C.VkRenderingAttachmentInfo)(C.calloc(1, C.size_t(unsafe.Sizeof(C.VkRenderingAttachmentInfo{}))))
		defer C.free(unsafe.Pointer(depth))
		fillAttachment(depth, ri.DepthAttachment)
		info.pDepthAttachment = depth
	}

	C.vkCmdBeginRendering(C.VkCommandBuffer(unsafe.Pointer(cb)), &info)
}

func CmdEndRendering(cb CommandBuffer) {
	C.vkCmdEndRendering(C.VkCommandBuffer(unsafe.Pointer(cb)))
}

// ---- dynamic state + draw ------------------------------------------------

func CmdSetViewport(cb CommandBuffer, vp Viewport) {
	v := C.VkViewport{
		x:        C.float(vp.X),
		y:        C.float(vp.Y),
		width:    C.float(vp.Width),
		height:   C.float(vp.Height),
		minDepth: C.float(vp.MinDepth),
		maxDepth: C.float(vp.MaxDepth),
	}
	C.vkCmdSetViewport(C.VkCommandBuffer(unsafe.Pointer(cb)), 0, 1, &v)
}

func CmdSetScissor(cb CommandBuffer, r Rect2D) {
	rect := C.VkRect2D{
		offset: C.VkOffset2D{x: C.int32_t(r.Offset.X), y: C.int32_t(r.Offset.Y)},
		extent: C.VkExtent2D{width: C.uint32_t(r.Extent.Width), height: C.uint32_t(r.Extent.Height)},
	}
	C.vkCmdSetScissor(C.VkCommandBuffer(unsafe.Pointer(cb)), 0, 1, &rect)
}

func CmdBindPipeline(cb CommandBuffer, bindPoint PipelineBindPoint, p Pipeline) {
	C.vkCmdBindPipeline(C.VkCommandBuffer(unsafe.Pointer(cb)),
		C.VkPipelineBindPoint(bindPoint), C.VkPipeline(unsafe.Pointer(p)))
}

func CmdBindDescriptorSets(cb CommandBuffer, bindPoint PipelineBindPoint, layout PipelineLayout, firstSet uint32, sets []DescriptorSet) {
	if len(sets) == 0 {
		return
	}
	C.vkCmdBindDescriptorSets(C.VkCommandBuffer(unsafe.Pointer(cb)),
		C.VkPipelineBindPoint(bindPoint), C.VkPipelineLayout(unsafe.Pointer(layout)),
		C.uint32_t(firstSet), C.uint32_t(len(sets)),
		(*C.VkDescriptorSet)(unsafe.Pointer(&sets[0])), 0, nil)
}

func CmdBindVertexBuffer(cb CommandBuffer, binding uint32, b Buffer, offset uint64) {
	vb := C.VkBuffer(unsafe.Pointer(b))
	off := C.VkDeviceSize(offset)
	C.vkCmdBindVertexBuffers(C.VkCommandBuffer(unsafe.Pointer(cb)), C.uint32_t(binding), 1, &vb, &off)
}

func CmdBindIndexBuffer(cb CommandBuffer, b Buffer, offset uint64, indexType IndexType) {
	C.vkCmdBindIndexBuffer(C.VkCommandBuffer(unsafe.Pointer(cb)),
		C.VkBuffer(unsafe.Pointer(b)), C.VkDeviceSize(offset), C.VkIndexType(indexType))
}

// CmdPushConstants uploads size bytes from data at the given offset. data must
// point at least size bytes (e.g. &myStruct or &slice[0]).
func CmdPushConstants(cb CommandBuffer, layout PipelineLayout, stages ShaderStageFlags, offset, size uint32, data unsafe.Pointer) {
	C.vkCmdPushConstants(C.VkCommandBuffer(unsafe.Pointer(cb)),
		C.VkPipelineLayout(unsafe.Pointer(layout)), C.VkShaderStageFlags(stages),
		C.uint32_t(offset), C.uint32_t(size), data)
}

func CmdDrawIndexed(cb CommandBuffer, indexCount, instanceCount, firstIndex uint32, vertexOffset int32, firstInstance uint32) {
	C.vkCmdDrawIndexed(C.VkCommandBuffer(unsafe.Pointer(cb)),
		C.uint32_t(indexCount), C.uint32_t(instanceCount), C.uint32_t(firstIndex),
		C.int32_t(vertexOffset), C.uint32_t(firstInstance))
}
