param(
    [string]$target = ""
)

# 设置环境变量
$env:NAME = "done-hub"
$env:DISTDIR = "dist"
$env:WEBDIR = "web"
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"

# 构建函数 - 需要在调用之前定义
function Invoke-BuildFunction {
    param([string]$functionName)

    switch ($functionName) {
        "all" { Build-All }
        "web" { Build-Web }
        "one-api" { Build-OneApi }
        "app" { Build-App }
        "clean" { Clean-Build }
        default {
            Write-Host "未知的目标: $functionName" -ForegroundColor Red
            Write-Host "可用的目标: all, web, one-api, app, clean" -ForegroundColor Yellow
            exit 1
        }
    }
}

function Build-All {
    Build-OneApi
}

function Build-Web {
    Write-Host "构建Web资源..." -ForegroundColor Cyan

    if (-not (Test-Path $env:WEBDIR)) {
        Write-Host "Web目录不存在!" -ForegroundColor Red
        exit 1
    }

    Set-Location $env:WEBDIR

    if (-not (Test-Path "build")) {
        New-Item -ItemType Directory -Path "build" | Out-Null
    }

    $env:VITE_APP_VERSION = $VERSION

    Write-Host "运行 yarn build..." -ForegroundColor Yellow
    yarn run build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "构建Web项目失败!" -ForegroundColor Red
        Set-Location ..
        exit 1
    }

    Set-Location ..
    Write-Host "Web资源构建完成" -ForegroundColor Green
}

function Build-OneApi {
    Build-Web

    Write-Host "当前 CGO_ENABLED:" -ForegroundColor Yellow
    go env CGO_ENABLED

    Write-Host "当前 GOOS:" -ForegroundColor Yellow
    go env GOOS

    Write-Host "当前 GOARCH:" -ForegroundColor Yellow
    go env GOARCH

    Write-Host "构建Go二进制文件..." -ForegroundColor Cyan

    if (-not (Test-Path $env:DISTDIR)) {
        New-Item -ItemType Directory -Path $env:DISTDIR | Out-Null
    }

    $outputPath = Join-Path $env:DISTDIR $env:NAME

    Write-Host "执行 go build..." -ForegroundColor Yellow
    go build -ldflags "-s -w -X 'done-hub/common/config.Version=$VERSION'" -o $outputPath

    if ($LASTEXITCODE -ne 0) {
        Write-Host "Go构建失败!" -ForegroundColor Red
        exit 1
    }

    Write-Host "构建完成: $outputPath" -ForegroundColor Green
}

function Build-App {
    Write-Host "当前 CGO_ENABLED:" -ForegroundColor Yellow
    go env CGO_ENABLED

    Write-Host "当前 GOOS:" -ForegroundColor Yellow
    go env GOOS

    Write-Host "当前 GOARCH:" -ForegroundColor Yellow
    go env GOARCH

    Write-Host "构建Go二进制文件..." -ForegroundColor Cyan

    if (-not (Test-Path $env:DISTDIR)) {
        New-Item -ItemType Directory -Path $env:DISTDIR | Out-Null
    }

    $outputPath = Join-Path $env:DISTDIR $env:NAME

    Write-Host "执行 go build..." -ForegroundColor Yellow
    go build -ldflags "-s -w -X 'done-hub/common/config.Version=$VERSION'" -o $outputPath

    if ($LASTEXITCODE -ne 0) {
        Write-Host "Go构建失败!" -ForegroundColor Red
        exit 1
    }

    Write-Host "构建完成: $outputPath" -ForegroundColor Green
}

function Clean-Build {
    Write-Host "清理构建产物..." -ForegroundColor Cyan

    if (Test-Path $env:DISTDIR) {
        Remove-Item -Path $env:DISTDIR -Recurse -Force
        Write-Host "已删除 $env:DISTDIR 目录" -ForegroundColor Green
    } else {
        Write-Host "$env:DISTDIR 目录不存在" -ForegroundColor Yellow
    }

    # 清理web/build目录
    $webBuildPath = Join-Path $env:WEBDIR "build"
    if (Test-Path $webBuildPath) {
        Remove-Item -Path $webBuildPath -Recurse -Force
        Write-Host "已删除 $webBuildPath 目录" -ForegroundColor Green
    }
}

# 获取版本信息，如果git describe失败则设置为dev
try {
    $VERSION = git describe --tags 2>$null
    if ($LASTEXITCODE -ne 0) {
        $VERSION = "dev"
    }
} catch {
    $VERSION = "dev"
}

Write-Host "当前版本:" -ForegroundColor Green
Write-Host $VERSION

# 如果没有指定目标，默认执行all
if ([string]::IsNullOrEmpty($target)) {
    Invoke-BuildFunction "all"
} else {
    Invoke-BuildFunction $target
}

exit $LASTEXITCODE

