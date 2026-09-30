#version 330

// The cloth of Victoria 3's fancy flag, lit the way the game lights it.
//
// The game runs its own physically based lighting over the cloth, with the
// sun of gfx/map/environment/ui_flag_environment.txt and an environment
// cubemap for what the sky adds. What is here is that in short: the coat of
// arms multiplied into the cloth, a normal map for the weave and the folds,
// one sun, one ambient term, and the colour adjustments the game ends on.

in vec2 fragTexCoord;
in vec2 fragTexCoord2;
in vec3 fragPosition;
in vec3 fragNormal;
in vec3 fragTangent;
in vec3 fragBitangent;

// The cloth's own three maps, which raylib binds in this order.
uniform sampler2D texture0; // diffuse: the weave and the shading painted into it
uniform sampler2D texture1; // properties: roughness in alpha, metalness in green
uniform sampler2D texture2; // normal: the folds of the weave

// The coat of arms, drawn into a target of its own and handed here.
uniform sampler2D flagTexture;

// Where the light comes from, from the game's environment file, mirrored
// across the x axis with the cloth itself so that it falls on the flag from
// the side the game lights it from.
const vec3 sunDirection = normalize(vec3(0.33, 1.0, -0.9));
const vec3 sunColor = vec3(1.0);
const float sunStrength = 1.6;
const float ambientStrength = 0.35;

// The light is worked out in linear light, the way the game does it, and only
// turned back into the values a screen shows at the end.
const float gamma = 2.2;

// The adjustments the game's shader ends on.
const float saturationScale = 0.88;
const vec3 tint = vec3(1.03, 0.77, 0.74);

out vec4 finalColor;

// The games pack a normal map with x in the red and alpha channels, which
// keeps its precision through the compression, and y in the green.
vec3 unpackNormal(vec4 sample)
{
    vec2 xy = vec2(sample.r * sample.a, sample.g) * 2.0 - 1.0;

    return vec3(xy, sqrt(clamp(1.0 - dot(xy, xy), 0.0, 1.0)));
}

vec3 toHSV(vec3 color)
{
    float high = max(color.r, max(color.g, color.b));
    float low = min(color.r, min(color.g, color.b));
    float chroma = high - low;

    float hue = 0.0;
    if (chroma > 0.0)
    {
        if (high == color.r)
        {
            hue = mod((color.g - color.b) / chroma, 6.0);
        }
        else if (high == color.g)
        {
            hue = (color.b - color.r) / chroma + 2.0;
        }
        else
        {
            hue = (color.r - color.g) / chroma + 4.0;
        }
    }

    return vec3(hue / 6.0, high > 0.0 ? chroma / high : 0.0, high);
}

vec3 toRGB(vec3 hsv)
{
    vec3 rgb = clamp(abs(mod(hsv.x * 6.0 + vec3(0.0, 4.0, 2.0), 6.0) - 3.0) - 1.0, 0.0, 1.0);

    return hsv.z * mix(vec3(1.0), rgb, hsv.y);
}

void main()
{
    vec4 cloth = texture(texture0, fragTexCoord);

    // The cloth's own picture says where the flag is: the pole and the air
    // around it are not part of it.
    if (cloth.a < 0.01)
    {
        discard;
    }

    vec4 properties = texture(texture1, fragTexCoord);
    // The coat of arms comes from a render target, which OpenGL fills from
    // the bottom up, so it is read the other way round.
    vec3 coatOfArms = texture(flagTexture, vec2(fragTexCoord2.x, 1.0 - fragTexCoord2.y)).rgb;

    vec3 albedo = pow(cloth.rgb * coatOfArms, vec3(gamma));

    mat3 tangentSpace = mat3(normalize(fragTangent), normalize(fragBitangent), normalize(fragNormal));
    vec3 normal = normalize(tangentSpace * unpackNormal(texture(texture2, fragTexCoord)));

    // One sun and what the sky adds, which is all a flag hanging in the
    // interface is lit by.
    float sun = max(dot(normal, sunDirection), 0.0);
    vec3 light = vec3(ambientStrength) + sunColor * sunStrength * sun;

    // A soft highlight along the folds, dulled by how rough the weave is.
    vec3 towardsViewer = normalize(-fragPosition);
    vec3 halfway = normalize(sunDirection + towardsViewer);
    float smoothness = clamp(properties.a, 0.0, 1.0);
    float highlight = pow(max(dot(normal, halfway), 0.0), mix(2.0, 48.0, smoothness)) * smoothness * sun;

    vec3 color = albedo * light + sunColor * highlight * 0.35;

    // The game takes a little of the colour out and warms what is left, then
    // turns the light into what a screen shows.
    vec3 hsv = toHSV(clamp(color, 0.0, 1.0));
    hsv.y *= saturationScale;
    color = pow(clamp(toRGB(hsv) * tint, 0.0, 1.0), vec3(1.0 / gamma));

    finalColor = vec4(color, cloth.a);
}
