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

// Colored emblem masks encode a per-pixel shading value in the blue
// channel instead of part of the color to match (0.0 = darkest,
// ~0.502 = neutral/1:1, 1.0 = lightest). When true, only red/green are
// used to identify which color slot a pixel belongs to.
uniform bool blueChannelShading;

out vec4 finalColor;

// 128/255: the neutral shading value where the replacement color is used as-is.
const float NEUTRAL_SHADE = 0.50196078;

vec3 applyBlueShading(vec3 replacement, float shade)
{
    if (shade < NEUTRAL_SHADE)
    {
        return mix(vec3(0.0), replacement, shade / NEUTRAL_SHADE);
    }

    return mix(
        replacement,
        vec3(1.0),
        (shade - NEUTRAL_SHADE) / (1.0 - NEUTRAL_SHADE)
    );
}

void main()
{
    vec4 texel = texture(texture0, fragTexCoord) * fragColor;

    for (int i = 0; i < MAX_RECOLORS; i++)
    {
        // Avoid reading inactive mapping slots.
        if (i >= recolorCount) break;

        float colorDistance = blueChannelShading
            ? distance(texel.rg, sourceColors[i].rg)
            : distance(texel.rgb, sourceColors[i]);

        // A smooth threshold avoids noisy hard edges.
        float matchAmount = 1.0 -
            smoothstep(tolerance * 0.75, tolerance, colorDistance);

        if (matchAmount > 0.0)
        {
            vec3 replacement;

            if (blueChannelShading)
            {
                replacement = applyBlueShading(replacementColors[i], texel.b);
            }
            else
            {
                float brightness = dot(texel.rgb, vec3(0.299, 0.587, 0.114));

                vec3 shadedReplacement = replacementColors[i] * brightness;
                replacement = mix(
                    replacementColors[i],
                    shadedReplacement,
                    preserveShading
                );
            }

            texel.rgb = mix(texel.rgb, replacement, matchAmount);

            // First matching mapping wins.
            break;
        }
    }

    finalColor = texel;
}