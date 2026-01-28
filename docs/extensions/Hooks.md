
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
      "delegator": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "cycle": 1,
      "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "kind": "delegator reward",
      "tx_kind": "fa1",
      "fa_token_id": "10",
      "fa_contract": "KT18amZmM5W7qDWVt2pH6uj7sCEd3kbzLrHT",
      "note": "reason"
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
  "cycle": 1,
  "payouts": [
    {
      "baker": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "delegator": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "cycle": 1,
      "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "kind": "delegator reward",
      "tx_kind": "fa1",
      "fa_token_id": "10",
      "fa_contract": "KT18amZmM5W7qDWVt2pH6uj7sCEd3kbzLrHT",
      "delegator_balance": "1000000000",
      "staked_balance": "1000000000",
      "amount": "1000000000",
      "fee_rate": 5,
      "fee": "1000000000",
      "note": "reason"
    }
  ],
  "own_staked_balance": "1000000000",
  "own_delegated_balance": "0",
  "external_staked_balance": "0",
  "external_delegated_balance": "0",
  "cycle_earned_fees": "1000000000",
  "cycle_earned_rewards": "1000000000",
  "cycle_earned_total": "2000000000",
  "bond_income": "1000000000",
  "fee_income": "0",
  "total_income": "0",
  "donated_bonds": "1000000000",
  "donated_fees": "0",
  "donated_total": "0",
  "timestamp": "0001-01-01T00:00:00Z"
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
      "kind": "delegator reward",
      "tx_kind": "fa1",
      "fa_token_id": "10",
      "fa_contract": "KT18amZmM5W7qDWVt2pH6uj7sCEd3kbzLrHT",
      "delegator_balance": "1000000000",
      "staked_balance": "1000000000",
      "amount": "1000000000",
      "fee_rate": 5,
      "fee": "1000000000",
      "note": "reason"
    }
  ],
  "invalid_payouts": null,
  "reports_of_past_succesful_payouts": [
    {
      "id": "7c7E1tgHsd48EmiM",
      "baker": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "timestamp": "2026-01-28T18:41:06.959342687Z",
      "cycle": 1,
      "kind": "delegator reward",
      "tx_kind": "fa1",
      "contract": "KT18amZmM5W7qDWVt2pH6uj7sCEd3kbzLrHT",
      "token_id": "10",
      "delegator": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "delegator_balance": "1000000000",
      "staked_balance": "1000000000",
      "recipient": "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU",
      "amount": "1000000000",
      "fee_rate": 5,
      "fee": "1000000000",
      "op_hash": "oneDGhZacw99EEFaYDTtWfz5QEhUW3PPVFsHa7GShnLPuDn7gSd",
      "success": true,
      "note": "reason"
    }
  ]
}
```

