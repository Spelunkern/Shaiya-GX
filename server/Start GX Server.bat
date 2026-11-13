@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0tools\start_server.ps1" %*
