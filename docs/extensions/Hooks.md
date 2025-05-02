
NOTE: *all bellow examples are just sample data to showcase fields used in data passed to hooks.*

## after_candidates_generated

This hook is capable of mutating data.
```json
{
  "cycle": 580,
  "candidates": [
    {
      "source": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "fee_rate": 5,
      "staked_balance": "1000000000",
      "delegated_balance": "1000000000",
      "is_invalid": true,
      "is_emptied": true,
      "is_baker_paying_tx_fee": true,
      "is_baker_paying_allocation_tx_fee": true,
      "invalid_because": "reason"
    }
  ]
}
```

## after_bonds_distributed

This hook is capable of mutating data.
```json
{
  "cycle": 580,
  "candidates": [
    {
      "source": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "fee_rate": 5,
      "staked_balance": "1000000000",
      "delegated_balance": "1000000000",
      "is_invalid": true,
      "is_emptied": true,
      "is_baker_paying_tx_fee": true,
      "is_baker_paying_allocation_tx_fee": true,
      "invalid_because": "reason",
      "bonds_amount": "1000000000",
      "tx_kind": "fa1",
      "fa_token_id": "10",
      "fa_contract": "KT18amZmM5W7qDWVt2pH6uj7sCEd3kbzLrHT"
    }
  ]
}
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
      "staked_balance": "1000000000",
      "delegated_balance": "1000000000",
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
}
```

## on_fees_collection

This hook is capable of mutating data.
```json
{
  "cycle": 580,
  "candidates": [
    {
      "source": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "fee_rate": 5,
      "staked_balance": "1000000000",
      "delegated_balance": "1000000000",
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
}
```

## after_payouts_blueprint_generated

This hook is NOT capable of mutating data *currently*.
```json
{
  "cycles": 1,
  "payouts": [
    {
      "baker": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "delegator": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "cycle": 1,
      "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "kind": "invalid",
      "tx_kind": "fa1",
      "fa_token_id": "10",
      "fa_contract": "KT18amZmM5W7qDWVt2pH6uj7sCEd3kbzLrHT",
      "delegator_balance": "1000000000",
      "amount": "1000000000",
      "fee_rate": 5,
      "fee": "1000000000",
      "op_limits": {
        "transaction_fee": 1,
        "storage_limit": 1,
        "gas_limit": 1,
        "deserialization_gas_limit": 1,
        "allocation_burn": 1,
        "storage_burn": 1
      },
      "note": "reason"
    }
  ],
  "summary": {
    "cycle": 1,
    "delegators": 2,
    "paid_delegators": 1,
    "own_staked_balance": "1000000000",
    "own_delegated_balance": "0",
    "external_staked_balance": "0",
    "external_delegated_balance": "0",
    "cycle_fees": "1000000000",
    "cycle_rewards": "1000000000",
    "distributed_rewards": "1000000000",
    "bond_income": "1000000000",
    "fee_income": "1000000000",
    "total_income": "1000000000",
    "donated_bonds": "1000000000",
    "donated_fees": "1000000000",
    "donated_total": "1000000000",
    "timestamp": "2023-01-01T00:00:00Z"
  }
}
```

## after_payouts_prepared

This hook is capable of mutating data *currently*.
```json
{
  "recipes": null,
  "payouts": [
    {
      "baker": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "delegator": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "cycle": 1,
      "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "kind": "invalid",
      "tx_kind": "fa1",
      "fa_token_id": "10",
      "fa_contract": "KT18amZmM5W7qDWVt2pH6uj7sCEd3kbzLrHT",
      "delegator_balance": "1000000000",
      "amount": "1000000000",
      "fee_rate": 5,
      "fee": "1000000000",
      "op_limits": {
        "transaction_fee": 1,
        "storage_limit": 1,
        "gas_limit": 1,
        "deserialization_gas_limit": 1,
        "allocation_burn": 1,
        "storage_burn": 1
      },
      "note": "reason"
    }
  ],
  "invalid_payouts": null,
  "reports_of_past_succesful_payouts": [
    {
      "id": "fd8d0230c9d70458",
      "baker": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "timestamp": "2025-05-02T13:14:47.129686485Z",
      "cycle": 1,
      "kind": "invalid",
      "tx_kind": "fa1",
      "contract": "KT18amZmM5W7qDWVt2pH6uj7sCEd3kbzLrHT",
      "token_id": "10",
      "delegator": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "delegator_balance": "1000000000",
      "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "amount": "1000000000",
      "fee_rate": 5,
      "fee": "1000000000",
      "tx_fee": 1,
      "op_hash": "oneDGhZacw99EEFaYDTtWfz5QEhUW3PPVFsHa7GShnLPuDn7gSd",
      "success": true,
      "note": "reason"
    }
  ]
}
```

