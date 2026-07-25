#define BCDEC_IMPLEMENTATION
#include "bcdec.h"

void bc7_decode_rgba8(
    const unsigned char *compressed_block,
    unsigned char *rgba_out,
    int destination_pitch_bytes)
{
    bcdec_bc7(compressed_block, rgba_out, destination_pitch_bytes);
}