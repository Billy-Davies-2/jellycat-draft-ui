#!/bin/bash
set -e

echo "Building Jellycat Fantasy Draft..."

# Build Go application
echo "Building Go binary..."
go build -o jellycat-draft main.go

# Compile TailwindCSS if CLI exists
if [ -f "./tailwindcss-linux-x64" ]; then
    echo "Compiling TailwindCSS..."
    ./tailwindcss-linux-x64 -c tailwind.config.go.js -i static/css/input.css -o static/css/styles.css --minify
else
    echo "TailwindCSS CLI not found, skipping CSS compilation"
fi

echo "Build complete! Run with: ./jellycat-draft"
