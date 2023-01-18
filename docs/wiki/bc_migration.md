# Migrating from BC

⚠️⚠️ **ledger wallet mode is not supported by `tezpay` yet** ⚠️⚠️

`tezpay` is able to build its config from preexisting BC configuration. So all you have to do is to use your old BC config and let `tezpay` to migrate it.

Your configuration gets migrated automatically on `pay` or `generate-payouts`. Old BC configuration will be saved to `config.backup.hjson`

You can run `tezpay` same way as BC - `tezpay pay --cycle=540` or `tezpay pay` for last completed cycle 😉

NOTE: *During BC migration `tezpay` injects 5% donation to your new `config.hjson` to support `tezpay` development. This is entirely optional. Set it as you see fit.*

## If you operate remote signer

`tezpay` does not touch configuration of your signers. To use remote signer with `tezpay` you have to change `public_key` to `pkh` in your `remote_signer.hjson`

For example:
```hjson
public_key: tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM
url: http://127.0.0.1:2222
```
becomes:
```hjson
pkh: tz1P6WKJu2rcbxKiKRZHKQKmKrpC9TfW1AwM
url: http://127.0.0.1:2222
```