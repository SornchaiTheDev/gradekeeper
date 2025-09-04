@echo off
echo Building GradeKeeper Cross-Platform...

REM Create dist directory
if not exist dist mkdir dist

echo.
echo Building Master Server...
cd master
go mod tidy

REM Build master for different platforms
echo   - Linux (amd64)
set GOOS=linux
set GOARCH=amd64
go build -o ../dist/gradekeeper-master-linux-amd64 main.go

echo   - Windows (amd64)
set GOOS=windows
set GOARCH=amd64
go build -o ../dist/gradekeeper-master-windows-amd64.exe main.go

echo   - macOS (amd64)
set GOOS=darwin
set GOARCH=amd64
go build -o ../dist/gradekeeper-master-darwin-amd64 main.go

echo   - macOS (arm64)
set GOOS=darwin
set GOARCH=arm64
go build -o ../dist/gradekeeper-master-darwin-arm64 main.go

cd ..

echo.
echo Building Cross-Platform Client...
go mod tidy

REM Build client for different platforms
echo   - Linux (amd64)
set GOOS=linux
set GOARCH=amd64
go build -o dist/gradekeeper-client-linux-amd64 client-crossplatform.go

echo   - Windows (amd64)
set GOOS=windows
set GOARCH=amd64
go build -o dist/gradekeeper-client-windows-amd64.exe client-crossplatform.go

echo   - macOS (amd64)
set GOOS=darwin
set GOARCH=amd64
go build -o dist/gradekeeper-client-darwin-amd64 client-crossplatform.go

echo   - macOS (arm64)
set GOOS=darwin
set GOARCH=arm64
go build -o dist/gradekeeper-client-darwin-arm64 client-crossplatform.go

echo.
echo Building Cross-Platform Standalone...

REM Build standalone for different platforms
echo   - Linux (amd64)
set GOOS=linux
set GOARCH=amd64
go build -o dist/gradekeeper-standalone-linux-amd64 standalone-crossplatform.go

echo   - Windows (amd64)
set GOOS=windows
set GOARCH=amd64
go build -o dist/gradekeeper-standalone-windows-amd64.exe standalone-crossplatform.go

echo   - macOS (amd64)
set GOOS=darwin
set GOARCH=amd64
go build -o dist/gradekeeper-standalone-darwin-amd64 standalone-crossplatform.go

echo   - macOS (arm64)
set GOOS=darwin
set GOARCH=arm64
go build -o dist/gradekeeper-standalone-darwin-arm64 standalone-crossplatform.go

echo.
echo Build complete! Generated files in dist/:
dir dist

echo.
echo Platform-specific executables:
echo Linux:   ./dist/gradekeeper-*-linux-amd64
echo Windows: ./dist/gradekeeper-*-windows-amd64.exe
echo macOS:   ./dist/gradekeeper-*-darwin-amd64 (Intel)
echo          ./dist/gradekeeper-*-darwin-arm64 (Apple Silicon)
pause