#version 330

// The cloth of Victoria 3's fancy flag, waved the way the game waves it.
//
// The maths is the game's own, from the vertex stage of its gui_flag_3d
// shader: a sine running along the flag away from the pole, pushing the cloth
// mostly away from the viewer, with the normal turned by the slope of that
// wave so that the light travels along the folds. The three numbers the game
// keeps in its engine rather than its files are set here to what looks like
// the flags in its interface.

in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec2 vertexTexCoord2;
in vec3 vertexNormal;
in vec4 vertexTangent;

uniform mat4 mvp;
uniform mat4 matModel;
uniform mat4 matNormal;

// time is what makes the flag wave: the seconds the window has been open.
uniform float time;

out vec2 fragTexCoord;
out vec2 fragTexCoord2;
out vec3 fragPosition;
out vec3 fragNormal;
out vec3 fragTangent;
out vec3 fragBitangent;

// The size of the cloth, which the wave is worked out in fractions of.
const vec2 clothSize = vec2(9.0, 6.0);

// Which way the wave pushes the cloth: mostly away from the viewer, a little
// upwards.
const vec3 waveDirection = vec3(0.0, 0.08, -1.0);

// How far the cloth swings, how many waves run along it, and how fast.
const float waveScale = 0.9;
const float smallWaveScale = 6.0;
const float animationSpeed = 1.6;

void main()
{
    // The wave runs from the pole, where the cloth is held and cannot move,
    // to the far edge, where it swings the most.
    float seed = clamp(vertexPosition.x / clothSize.x + 0.5, 0.0, 1.0);

    float phase = time * animationSpeed - seed * smallWaveScale;
    float wave = waveScale * smoothstep(0.0, 0.12, seed) * sin(phase);
    float slope = waveScale * seed * -(sin(phase) + cos(phase) * -(seed * smallWaveScale));

    vec3 position = vertexPosition + waveDirection * wave;

    // The normal of the wave itself, which the cloth's own normal is turned
    // most of the way towards.
    vec2 alongWave = normalize(vec2(1.0, slope));
    vec3 waveNormal = normalize(vec3(alongWave.y, 0.0, -alongWave.x));
    vec3 normal = normalize(mix(vertexNormal, waveNormal, 0.65));

    fragTexCoord = vertexTexCoord;
    fragTexCoord2 = vertexTexCoord2;
    fragPosition = vec3(matModel * vec4(position, 1.0));
    fragNormal = normalize(vec3(matNormal * vec4(normal, 0.0)));
    fragTangent = normalize(vec3(matNormal * vec4(vertexTangent.xyz, 0.0)));

    // The fourth number of a tangent says which way round the bitangent goes.
    fragBitangent = cross(fragNormal, fragTangent) * vertexTangent.w;

    gl_Position = mvp * vec4(position, 1.0);
}
