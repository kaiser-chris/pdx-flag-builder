mkdir bin\
mkdir bin\windows
odin build src -define:RELEASE=true -o:speed -subsystem:windows -out:bin/windows/pdx-flag-builder.exe -resource:resources.rc
robocopy assets\ bin\windows\assets\ /E /NJH /NJS
del resources.res
cd bin\windows
del "pdx-flag-builder_x.x.x_windows_amd64.zip"
tar.exe --exclude="*.zip" -a -cf "pdx-flag-builder_x.x.x_windows_amd64.zip" "*"
cd ..\..