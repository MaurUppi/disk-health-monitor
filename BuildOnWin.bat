@echo off
setlocal

REM Get current date and time
for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /format:list') do set datetime=%%I

REM Extract year, month, day, hour, minute, second
set year=%datetime:~0,4%
set month=%datetime:~4,2%
set day=%datetime:~6,2%
set hour=%datetime:~8,2%
set minute=%datetime:~10,2%
set second=%datetime:~12,2%

REM Format as ISO 8601 (YYYY-MM-DDTHH:MM:SSZ)
set BUILD_DATE=%year%-%month%-%day%T%hour%:%minute%:%second%Z

REM Set version based on date
set VERSION=%year%-%month%-%day%

REM Set app name and main package
set APP_NAME=disk-health-monitor
set MAIN_PACKAGE=./cmd/monitor

REM Display build info
echo Building %APP_NAME% version %VERSION% with build date %BUILD_DATE%...

REM Build in Docker with environment variables
docker run --rm -v "%CD%":/app -w /app -e VERSION="%VERSION%" -e BUILD_DATE="%BUILD_DATE%" go-linux-builder sh -c "export LDFLAGS=\"-s -w -X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE}\"; echo \"go build -buildvcs=false -ldflags '${LDFLAGS}' -o %APP_NAME% %MAIN_PACKAGE%\"; go build -buildvcs=false -ldflags \"${LDFLAGS}\" -o %APP_NAME% %MAIN_PACKAGE%"

REM Check build result
if %ERRORLEVEL% EQU 0 (
    echo Build successful!
    echo Binary location: %CD%\%APP_NAME%
) else (
    echo Build failed with error code %ERRORLEVEL%
)
endlocal