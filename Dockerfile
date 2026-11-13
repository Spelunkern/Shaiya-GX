# syntax=docker/dockerfile:1
#
# Toolchain image for Shaiya GX. The source is mounted at /src and the output
# folder at /out, so the compiler writes the executable directly into the game
# folder configured in docker-compose.yml. Nothing is copied afterwards.
#
#   TARGET_OS=windows (default) -> game.exe  (CGO, mingw-w64 cross compile)
#   TARGET_OS=linux             -> shaiya    (CGO, GLFW/GL/X11/ALSA headers)
ARG GO_IMAGE=golang:1.27-alpine
FROM ${GO_IMAGE}

RUN apk add --no-cache \
        build-base pkgconf git mingw-w64-gcc \
        mesa-dev glfw-dev wayland-dev libxkbcommon-dev \
        libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev \
        alsa-lib-dev freetype-dev

# The window layer links GLFW and OpenGL, the audio layer links ALSA: CGO is
# mandatory in both targets.
ENV CGO_ENABLED=1
WORKDIR /src
