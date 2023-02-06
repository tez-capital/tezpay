# tezpay
- https://github.com/teambition/jsonrpc-go
## stdio
	- writeline/readline - one request/response/notification per line
## http/s
	- net/http posts to jsonrpc servers

# extensions
## GO
- github.com/osamingo/jsonrpc/v2@latest
- adjust for stdio 
  - requires serve stdio in https://github.com/osamingo/jsonrpc/blob/master/handler.go#L28
  - and parse and send in https://github.com/osamingo/jsonrpc/blob/74b8a654353fc5241c229b04a739eb8b9657df5d/jsonrpc.go#L40-L109