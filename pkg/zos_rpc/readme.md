# zos rpc api

### example

```bash
# request
curl -X POST http://192.168.123.48:3000/rpc \
-H "Content-Type: application/json" \
-d '{
  "jsonrpc": "2.0",
  "method": "system_version",
  "params": [],
  "id": 1
}'

# response
{"jsonrpc":"2.0","result":{"zos":"0.0.0","zinit":"v0.2.11"},"id":1}
```
