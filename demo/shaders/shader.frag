#version 450
#extension GL_EXT_nonuniform_qualifier : require

layout(location = 0) in vec3 inNormal;
layout(location = 1) in vec2 inUV;
layout(location = 2) in vec3 inWorldPos;
layout(location = 3) in vec3 inLightPos;
layout(location = 4) flat in uint inInstance;

layout(location = 0) out vec4 outColor;

// Bindless combined-image-sampler array (descriptor indexing). One texture per
// model instance, indexed non-uniformly by gl_InstanceIndex.
layout(set = 0, binding = 0) uniform sampler2D textures[];

void main() {
    vec3 N = normalize(inNormal);
    vec3 L = normalize(inLightPos - inWorldPos);
    float diff = max(dot(N, L), 0.0);
    vec3 tex = texture(textures[nonuniformEXT(inInstance)], inUV).rgb;
    vec3 color = tex * (0.2 + 0.8 * diff);
    outColor = vec4(color, 1.0);
}
