Material Symbols Outlined
=========================

From https://fonts.google.com/icons, licensed under the Apache License 2.0;
LICENSE.txt is the licence as it was downloaded with the font.

MaterialSymbolsOutlined-Subset.ttf is MaterialSymbolsOutlined-Regular.ttf cut
down to the icons the interface draws, which is why it is two kilobytes rather
than a megabyte. The whole font holds some four thousand icons, and the
application would carry every one of them around for the sake of a handful.

The icons in it, by the name the icon gallery knows them under and the code
point the font maps them to:

    arrow_downward  U+E5DB
    arrow_upward    U+E5D8
    content_copy    U+E14D
    delete          U+E872
    link            U+E157
    link_off        U+E16F

To add one, look its code point up in the gallery, download the font again and
cut it down with fonttools (pip install fonttools):

    pyftsubset MaterialSymbolsOutlined-Regular.ttf \
        --output-file=MaterialSymbolsOutlined-Subset.ttf \
        --unicodes=E14D,E157,E16F,E5D8,E5DB,E872,<the new one> \
        --layout-features= --name-IDs='*' --name-legacy --notdef-outline

The names are kept so that the copyright notice travels with the font, and the
layout features are dropped because the gallery's ligature names, which is all
they are there for, are of no use to an interface that asks for code points.
