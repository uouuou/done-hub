@echo off
setlocal enabledelayedexpansion

set NAME=done-hub
set DISTDIR=dist
set WEBDIR=web
SET CGO_ENABLED=0
SET GOOS=linux
SET GOARCH=amd64

rem 获取版本信息，如果git describe失败则设置为dev
for /f "delims=" %%a in ('git describe --tags 2^>nul') do set VERSION=%%a
if "%VERSION%"=="" set VERSION=dev

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

echo now the CGO_ENABLED:
 go env CGO_ENABLED

echo now the GOOS:
 go env GOOS

echo now the GOARCH:
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

echo now the CGO_ENABLED:
 go env CGO_ENABLED

echo now the GOOS:
 go env GOOS

echo now the GOARCH:
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
echo   one-api  Build Go binary only
echo   clean    Remove build artifacts
exit /b
