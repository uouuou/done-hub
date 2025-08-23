@echo off

set NAME=done-hub
set DISTDIR=dist
set WEBDIR=web
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64

rem Get version info, set to dev if git describe fails
for /f "delims=" %%a in ('git describe --tags 2^>nul') do set VERSION=%%a
if "%VERSION%"=="" set VERSION=dev

echo Current version:
echo %VERSION%

if "%1"=="" (
    call :all
) else (
    call :%1
)

exit /b %errorlevel%

:all
call :one-api
exit /b

:web
echo Building web resources...
if not exist %WEBDIR% (
    echo Web directory not found!
    exit /b 1
)

cd %WEBDIR%
if not exist %WEBDIR%\build (
    mkdir %WEBDIR%\build
)

set VITE_APP_VERSION=%VERSION%
call yarn run build
if errorlevel 1 (
    echo Failed to build web project!
    exit /b 1
)

cd ..
exit /b

:one-api
call :web

echo Current CGO_ENABLED:
go env CGO_ENABLED

echo Current GOOS:
go env GOOS

echo Current GOARCH:
go env GOARCH

echo Building Go binary...
if not exist %DISTDIR% mkdir %DISTDIR%

go build -ldflags "-s -w -X 'done-hub/common/config.Version=%VERSION%'" -o %DISTDIR%\%NAME%
if errorlevel 1 (
    echo Go build failed!
    exit /b 1
)

echo Build completed: %DISTDIR%\%NAME%
exit /b

:app

echo Current CGO_ENABLED:
go env CGO_ENABLED

echo Current GOOS:
go env GOOS

echo Current GOARCH:
go env GOARCH

echo Building Go binary...
if not exist %DISTDIR% mkdir %DISTDIR%

go build -ldflags "-s -w -X 'done-hub/common/config.Version=%VERSION%'" -o %DISTDIR%\%NAME%
if errorlevel 1 (
    echo Go build failed!
    exit /b 1
)

echo Build completed: %DISTDIR%\%NAME%
exit /b

:clean
echo Cleaning build artifacts...
if exist %DISTDIR% (
    rd /s /q %DISTDIR%
    echo Deleted %DISTDIR%
)

if exist %WEBDIR%\build (
    rd /s /q %WEBDIR%\build
    echo Deleted %WEBDIR%\build
)

exit /b

:help
echo Usage:
echo   build.bat [target]
echo Targets:
echo   all      Build everything (default)
echo   web      Build web resources
echo   one-api  Build Go binary (requires built web resources)
echo   app      Build Go binary only
echo   clean    Remove build artifacts
exit /b
