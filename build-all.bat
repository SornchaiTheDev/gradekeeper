@echo off
echo Building GradeKeeper Master and Client...

echo.
echo Building Master Server...
cd master
go mod tidy
set GOOS=windows
set GOARCH=amd64
go build -o gradekeeper-master.exe main.go
if %ERRORLEVEL% == 0 (
    echo Master server built successfully!
    move gradekeeper-master.exe ..
) else (
    echo Master server build failed!
)

echo.
echo Building Client...
cd ..
go mod tidy
set GOOS=windows
set GOARCH=amd64
go build -o gradekeeper-client.exe client.go
if %ERRORLEVEL% == 0 (
    echo Client built successfully!
) else (
    echo Client build failed!
)

echo.
echo Building Standalone Version...
go build -o gradekeeper-standalone.exe main.go
if %ERRORLEVEL% == 0 (
    echo Standalone version built successfully!
    echo.
    echo Build complete! You have:
    echo - gradekeeper-master.exe (Master server with web dashboard)
    echo - gradekeeper-client.exe (Client that connects to master)
    echo - gradekeeper-standalone.exe (Original standalone version)
) else (
    echo Standalone build failed!
)
pause