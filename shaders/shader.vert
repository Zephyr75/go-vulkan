#version 450
#extension GL_EXT_buffer_reference : require
#extension GL_EXT_scalar_block_layout : require

// Interleaved vertex input (obj.Vertex: pos, normal, uv).
layout(location = 0) in vec3 inPos;
layout(location = 1) in vec3 inNormal;
layout(location = 2) in vec2 inUV;

layout(location = 0) out vec3 outNormal;
layout(location = 1) out vec2 outUV;
layout(location = 2) out vec3 outWorldPos;
layout(location = 3) out vec3 outLightPos;
layout(location = 4) flat out uint outInstance;

// Per-frame scene data, reached through a buffer device address in the push
// constant (descriptor-free access, scalar layout to match the Go struct).
layout(buffer_reference, scalar) readonly buffer SceneData {
    mat4 projection;
    mat4 view;
    mat4 model[3];
    vec4 lightPos;
    uint selected;
};

layout(push_constant) uniform Push {
    SceneData scene;
} pc;

void main() {
    mat4 model = pc.scene.model[gl_InstanceIndex];
    vec4 world = model * vec4(inPos, 1.0);
    gl_Position = pc.scene.projection * pc.scene.view * world;
    outNormal = mat3(model) * inNormal;
    outUV = inUV;
    outWorldPos = world.xyz;
    outLightPos = pc.scene.lightPos.xyz;
    outInstance = gl_InstanceIndex;
}
