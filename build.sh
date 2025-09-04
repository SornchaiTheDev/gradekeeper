#!/bin/bash
echo "Building gradekeeper for Windows..."
GOOS=windows GOARCH=amd64 go build -o gradekeeper.exe main.go
if [ $? -eq 0 ]; then
    echo "Build successful! Transfer gradekeeper.exe to Windows and run it."
else
    echo "Build failed!"
fi