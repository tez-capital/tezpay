docs/cmd/tezpay.md## tezpay

TEZPAY

### Synopsis

TEZPAY dev - the tezos reward distributor
Copyright © 2025 alis.is


```
tezpay [flags]
```

### Options

```
      --disable-donation-prompt          Disable donation prompt
  -h, --help                             help for tezpay
      --log-file string                  Logs to file
  -l, --log-level string                 Sets log level format (trace/debug/info/warn/error) (default "info")
      --log-server string                launches log server at specified address
  -o, --output-format string             Sets output log format (json/text/auto) (default "auto")
  -p, --path string                      path to working directory (default ".")
      --pay-only-address-prefix string   Pays only to addresses starting with the prefix (e.g. KT, usually you do not want to use this, just for recovering in case of issues)
      --signer string                    Override signer
      --skip-version-check               Skip version check
      --version                          Prints version
```

### SEE ALSO

* [tezpay continual](/tezpay/reference/cmd/tezpay_continual)	 - continual payout
* [tezpay generate-payouts](/tezpay/reference/cmd/tezpay_generate-payouts)	 - generate payouts
* [tezpay import-configuration](/tezpay/reference/cmd/tezpay_import-configuration)	 - seed configuration from
* [tezpay pay](/tezpay/reference/cmd/tezpay_pay)	 - manual payout
* [tezpay pay-date-range](/tezpay/reference/cmd/tezpay_pay-date-range)	 - EXPERIMENTAL: payout for date range
* [tezpay statistics](/tezpay/reference/cmd/tezpay_statistics)	 - prints earning stats
* [tezpay test-extensions](/tezpay/reference/cmd/tezpay_test-extensions)	 - extensions test
* [tezpay test-notify](/tezpay/reference/cmd/tezpay_test-notify)	 - notification test
* [tezpay transfer](/tezpay/reference/cmd/tezpay_transfer)	 - transfers tez to specified address
* [tezpay version](/tezpay/reference/cmd/tezpay_version)	 - prints tezpay version

###### Auto generated by spf13/cobra on 25-Jul-2025
