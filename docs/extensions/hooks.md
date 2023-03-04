# Available Hooks

NOTE: *all bellow examples are just sample data to showcase fields used in data passed to hooks.*

## after_candidate_generated

This hook is capable of mutating data.
```json
[
  {
    "source": "",
    "recipient": "",
    "fee_rate": 5,
    "balance": "1000000000",
    "is_invalid": true,
    "is_emptied": true,
    "is_baker_paying_tx_fee": true,
    "is_baker_paying_allocation_tx_fee": true,
    "invalid_because": "reason"
  }
]
```

## after_bonds_distributed

This hook is capable of mutating data.
```json
[
  {
    "source": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
    "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
    "fee_rate": 5,
    "balance": "1000000000",
    "is_invalid": true,
    "is_emptied": true,
    "is_baker_paying_tx_fee": true,
    "is_baker_paying_allocation_tx_fee": true,
    "invalid_because": "reason",
    "bonds_amount": "1000000000",
    "tx_kind": "fa2",
    "fa_token_id": "10",
    "fa_contract": "KT18amZmM5W7qDWVt2pH6uj7sCEd3kbzLrHT"
  }
]
```

## check_balance

This hook is NOT capable of mutating data.
```json
{
  "skip_tez_check": true,
  "is_sufficient": true,
  "message": "This message is used to carry errors from hook to the caller.",
  "payouts": [
    {
      "source": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "fee_rate": 5,
      "balance": "1000000000",
      "is_invalid": true,
      "is_emptied": true,
      "is_baker_paying_tx_fee": true,
      "is_baker_paying_allocation_tx_fee": true,
      "invalid_because": "reason",
      "bonds_amount": "1000000000",
      "tx_kind": "tez",
      "fa_token_id": "10",
      "fa_contract": "KT18amZmM5W7qDWVt2pH6uj7sCEd3kbzLrHT"
    }
  ]
}
```

## on_fees_collection

This hook is capable of mutating data.
```json
[
  {
    "source": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
    "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
    "fee_rate": 5,
    "balance": "1000000000",
    "is_invalid": true,
    "is_emptied": true,
    "is_baker_paying_tx_fee": true,
    "is_baker_paying_allocation_tx_fee": true,
    "invalid_because": "reason",
    "bonds_amount": "1000000000",
    "tx_kind": "fa1",
    "fa_token_id": "10",
    "fa_contract": "KT18amZmM5W7qDWVt2pH6uj7sCEd3kbzLrHT",
    "fee": "1000000000"
  }
]
```

