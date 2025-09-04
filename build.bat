@echo off
echo Building gradekeeper for Windows...
set GOOS=windows
set GOARCH=amd64
go build -o gradekeeper.exe main.go
if %ERRORLEVEL% == 0 (
    echo Build successful! Run gradekeeper.exe to execute the program.
) else (
    echo Build failed!
)
pause