odin build src -define:RELEASE=true -o:speed -subsystem:windows -out:bin/pdx-flag-builder.exe -resource:resources.rc
robocopy assets\ bin\assets\ /E
del resources.res