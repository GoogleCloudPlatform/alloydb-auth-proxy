@echo off

setlocal

set SERVICE=alloydb-auth-proxy

net stop "%SERVICE%"
sc.exe delete "%SERVICE%"

endlocal
