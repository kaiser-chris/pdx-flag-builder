#version 330

#define MAX_RECOLORS 8

in vec2 fragTexCoord;
in vec4 fragColor;

uniform sampler2D texture0;

uniform int recolorCount;

// Each element is normalized RGB: 0.0 to 1.0.
uniform vec3 sourceColors[MAX_RECOLORS];
uniform vec3 replacementColors[MAX_RECOLORS];

// One tolerance for every mapping.
uniform float tolerance;

// 0.0 = exact replacement, 1.0 = preserve source brightness.
uniform float preserveShading;

out vec4 finalColor;

void main()
{
    vec4 texel = texture(texture0, fragTexCoord) * fragColor;

    for (int i = 0; i < MAX_RECOLORS; i++)
    {
        // Avoid reading inactive mapping slots.
        if (i >= recolorCount) break;

        float colorDistance = distance(texel.rgb, sourceColors[i]);

        // A smooth threshold avoids noisy hard edges.
        float matchAmount = 1.0 -
            smoothstep(tolerance * 0.75, tolerance, colorDistance);

        if (matchAmount > 0.0)
        {
            float brightness = dot(texel.rgb, vec3(0.299, 0.587, 0.114));

            vec3 shadedReplacement = replacementColors[i] * brightness;
            vec3 replacement = mix(
                replacementColors[i],
                shadedReplacement,
                preserveShading
            );

            texel.rgb = mix(texel.rgb, replacement, matchAmount);

            // First matching mapping wins.
            break;
        }
    }

    finalColor = texel;
}