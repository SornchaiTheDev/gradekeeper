#!/bin/bash
echo "Building GradeKeeper Master and Client..."

echo ""
echo "Building Master Server..."
cd master
go mod tidy
GOOS=windows GOARCH=amd64 go build -o gradekeeper-master.exe main.go
if [ $? -eq 0 ]; then
    echo "Master server built successfully!"
    mv gradekeeper-master.exe ..
else
    echo "Master server build failed!"
fi

echo ""
echo "Building Client..."
cd ..
go mod tidy
GOOS=windows GOARCH=amd64 go build -o gradekeeper-client.exe client.go
if [ $? -eq 0 ]; then
    echo "Client built successfully!"
else
    echo "Client build failed!"
fi

echo ""
echo "Building Standalone Version..."
GOOS=windows GOARCH=amd64 go build -o gradekeeper-standalone.exe main.go
if [ $? -eq 0 ]; then
    echo "Standalone version built successfully!"
    echo ""
    echo "Build complete! You have:"
    echo "- gradekeeper-master.exe (Master server with web dashboard)"
    echo "- gradekeeper-client.exe (Client that connects to master)"
    echo "- gradekeeper-standalone.exe (Original standalone version)"
else
    echo "Standalone build failed!"
fi