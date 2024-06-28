# Tezpay Payout Fixer extension

Compares the payouts reports with the payouts based on config and inject compensation transactions to fix the difference.

## Installation

1. Download the extension from the [releases page](https://github.com/tez-capital/tezpay/releases) based on your platform.
2. Place it into directory where you have tezpay installed.
3. Add the extension to your tezpay configuration
```yaml
...
	extensions: [
		{
			name: payout-fixer
			command: "./tezpay-payout-fixer" // or .exe based on your platform
			kind: stdio
			hooks: [
				after_payouts_prepared:rw
			]
		}
	]
...
```
1. Run pay `tezpay pay --cycle <cycle>`

## Notes

- Does not affect generation result. Actual changes are visible before the payout when you are prompted to confirm the payouts. (hint: you can use --dry-run)
- We do not recommend to run it in continual mode, use only when paying out manually.
- Creates log file `tezpay-fixer.log` where it lists all the changes made.
- Can be used to fix any kind of payout issue - e.g. missing payouts, wrong payouts after incorrect config, etc.