package main

// On Windows the C and C++ runtime libraries of MinGW are linked into the
// executable. libstdc++ (Dear ImGui is C++), libgcc and winpthreads come with
// MinGW, not with Windows, so an executable that loads them as DLLs only
// starts where a MinGW toolchain happens to be on PATH.
//
// Setting this here rather than on the command line makes it hold for every
// build: go build, the scripts, the release workflow and the Store package.
// link_test.go checks that the executable loads nothing but DLLs of Windows.

// #cgo windows LDFLAGS: -static
import "C"
