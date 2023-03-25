# TezPay Container Readme
TezPay is a Tezos reward distributor that simplifies the process of distributing rewards to your stakeholders. This readme provides instructions on how to use the [tez-capital/tezpay](ghcr.io/tez-capital/tezpay) container image, which comes with both [tezpay](https://github.com/tez-capital/tezpay) and [eli](https://github.com/alis-is/eli) preinstalled.

## Prerequisites
Docker installed on your system.
## Usage
1. Pull the TezPay container image:
```bash
docker pull ghcr.io/tez-capital/tezpay
```
2. Run the TezPay container with the desired command. Replace `[command]` with the desired TezPay command and `[options]` with the corresponding command options:
```bash
docker run --rm -it -v $(pwd):/tezpay ghcr.io/tez-capital/tezpay [command] [options]
```

Here are some examples of how to use the TezPay commands:

Generate payouts:
```bash
docker run --rm -it -v $(pwd):/tezpay ghcr.io/tez-capital/tezpay generate-payouts --cycle <cycle_number> [flags]
```
Replace `<cycle_number>` with the desired cycle number for which you want to generate payouts.

Continual payout (executed by default if no commands or arguments are provided):
```bash
docker run --rm -it -v $(pwd):/tezpay ghcr.io/tez-capital/tezpay continual [flags]
```

Manual payout:
```bash
docker run --rm -it -v $(pwd):/tezpay ghcr.io/tez-capital/tezpay pay --cycle <cycle_number> [flags]
```

**Note**: When running the container, make sure to mount the current working directory (or the desired directory containing your TezPay configuration) to the `/tezpay` path inside the container. This ensures that the container has access to your configuration files and can write any generated files back to your host system. Your TezPay configuration should be named `config.hjson`. Payout reports will be stored in the mounted directory under the `reports` directory by default.

For more information about available commands and their options, refer to the provided TezPay help:

```bash
docker run --rm -it ghcr.io/tez-capital/tezpay help
```

## Support
For any questions or issues related to the container or TezPay, please visit the GitHub repository at [tez-capital/tezpay](https://github.com/tez-capital/tezpay) or submit an issue there.
