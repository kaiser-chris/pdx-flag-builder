mkdir bin/
mkdir bin/linux
odin build src -define:RELEASE=true -o:speed -out:bin/linux/pdx-flag-builder
cp -r assets/ bin/linux/
cd bin/linux/
tar --exclude='*.zip' -a -cf 'pdx-flag-builder_x.x.x_windows_amd64.zip' *
cd ..\..