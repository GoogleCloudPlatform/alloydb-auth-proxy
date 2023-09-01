@echo off

setlocal

set SERVICE=alloydb-auth-proxy
set DISPLAYNAME=Google AlloyDB Auth Proxy
set CREDENTIALSFILE=%~dp0key.json
set INSTANCEURI=projects/<PROJECT>/locations/<REGION>/clusters/<CLUSTER>/instances/<INSTANCE>

sc.exe create "%SERVICE%" binPath= "\"%~dp0alloydb-auth-proxy.exe\" --credentials-file \"%CREDENTIALSFILE%\" %INSTANCEURI%" obj= "NT AUTHORITY\Network Service" start= auto displayName= "%DISPLAYNAME%"
sc.exe failure "%SERVICE%" reset= 0 actions= restart/0/restart/0/restart/0
net start "%SERVICE%"

endlocal
