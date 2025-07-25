docs/cmd/tezpay_transfer.md## tezpay transfer

transfers tez to specified address

### Synopsis

transfers tez to specified address from payout wallet

```
tezpay transfer <destination> <amount tez> [flags]
```

### Options

```
      --confirm   automatically confirms transfer
  -h, --help      help for transfer
      --mutez     amount in mutez
```

### Options inherited from parent commands

```
      --disable-donation-prompt          Disable donation prompt
      --log-file string                  Logs to file
  -l, --log-level string                 Sets log level format (trace/debug/info/warn/error) (default "info")
      --log-server string                launches log server at specified address
  -o, --output-format string             Sets output log format (json/text/auto) (default "auto")
  -p, --path string                      path to working directory (default ".")
      --pay-only-address-prefix string   Pays only to addresses starting with the prefix (e.g. KT, usually you do not want to use this, just for recovering in case of issues)
      --signer string                    Override signer
      --skip-version-check               Skip version check
```

### SEE ALSO

* [tezpay](/tezpay/reference/cmd/tezpay)	 - TEZPAY

###### Auto generated by spf13/cobra on 25-Jul-2025
