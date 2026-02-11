@echo off
setlocal
set "PATH=%PATH%;C:\Program Files\Go\bin"

cd /d %~dp0

if not exist .env (
  copy .env.example .env
)

cd /d %~dp0backend

"C:\Program Files\Go\bin\go.exe" mod tidy
if errorlevel 1 exit /b 1

cd /d %~dp0

docker compose -f deploy\docker-compose.yml up -d --build
if errorlevel 1 exit /b 1

cd /d %~dp0backend

"C:\Program Files\Go\bin\go.exe" run .\cmd\server migrate
if errorlevel 1 exit /b 1

"C:\Program Files\Go\bin\go.exe" run .\cmd\server seed
if errorlevel 1 exit /b 1

echo Deploy complete. UI: http://localhost:3000
