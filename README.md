# Cross Offcie VPN Client Service
Web and Websocket service for external gui application

## Web querys
-- check client version for copability
```
/check?version=х.х.х
```

# WSS querys
## Start query
-- Example of start query:
```
{
  "data": {
    "ID": 1,
    "CreatedAt": "2024-02-24T15:31:28.476277+07:00",
    "UpdatedAt": "2024-02-24T15:43:48.7720646+07:00",
    "DeletedAt": null,
    "Name": "sdf",
    "Endpoint": "wss://127.0.0.1:443",
    "User": "Usertest",
    "Password": "PaswdTest",
    "Sni": "",
    "SkipVerifySSL": true,
    "UseRemoteRouteRules": true,
    "LocalRouteRules": "",
    "ProxyDomains": ""
  },
  "event": "start"
}
```
--  Example answer of start query:
```
{
  "data": null,
  "event": "started"
}
```
-- If start successfully, server send start info: 
```
{
  "data": {
    "dns": "8.8.8.8",
    "ip": "10.8.57.69",
    "remoteIp": "127.0.0.1",
    "route": [
      "1.0.0.0/8"
    ]
  },
  "event": "allocated"
}
```
-- If start with error, server send error info
```
{
  "data": {
    "error": "connet fail,network error",
    "type": "network"
  },
  "event": "error"
}
```
## Stop query
– Example of stop query:
```
{
  "event": "stop"
}
```
– Example answer of stop query:
```
{
  "data": null,
  "event": "stoped"
}
```
## Statistics query
-- Example of statistics query:
```
{
  "event": "getbytes"
}
```
-- Example answer of statistics query:
```
{
  "data": {
    "DownBytes": 151,
    "UpBytes": 4
  },
  "event": "bytes"
}
```

