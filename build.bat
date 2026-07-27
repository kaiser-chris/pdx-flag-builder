odin build src -define:RELEASE=true -o:speed -subsystem:windows -out:bin/pdx-flag-builder.exe
robocopy assets\ bin\assets\ /E