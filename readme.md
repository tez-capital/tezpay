# TEZPAY
<p align="center"><img width="100" src="https://raw.githubusercontent.com/alis-is/tezpay/main/assets/logo.png" alt="TEZPAY logo"></p>

Heyπ I am PayBuddy close friend of your BakeBuddy π¨βπ³ you likely know. I am determined to provide you best experience with your baker payouts.
Because of that I prepared something special for you - TEZPAY.

See [Command Reference](docs/cmd) for details about commands. 

## Contributing

To contribute to TEZPAY please read [CONTRIBUTING.md](docs/CONTRIBUTING.md)

## Setup

1. Create directory where you want to store your `tezpay` configuration and reports
	- e.g. `mk tezpay`
2. Head to [Releases](https://github.com/alis-is/tezpay/releases) and download latest release and place it into newly created directory
	- on linux you can just `wget -q https://raw.githubusercontent.com/alis-is/tezpay/main/install.sh -O /tmp/install.sh && sh /tmp/install.sh`
3. Create and adjust configuration file `config.hjson`  See our configuration examples for all available options.
4. ...
5. Run `tezpay pay` to pay latest cycle

## Credits

- TEZPAY [default data collector](https://github.com/tez-capital/tezpay/blob/main/clients/default_colletor.go#L39) - only available right now - is **Powered by [TzKT API](https://api.tzkt.io/)**
