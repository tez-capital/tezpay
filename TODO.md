TODO:
- [x] fix summary
- [x] fix notification summary
- [ ] revisit report invalid payouts - there is probably better way to enforce we do not get accumulated reports than panic

- [ ] test generate
- [ ] test continual
- [ ] test pay
- [ ] test pay date range

Notes:
- invalid payouts are no longer included in the fee
- rather there are meant to be exposed as dangling in produced payout report
    * dangling is sum of amounts of invalid payouts
- this is because invalid in the age of aggregation may not be permanent. 