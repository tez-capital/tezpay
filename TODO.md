TODO:
- [x] fix summary
- [x] fix notification summary

- [ ] test generate
- [ ] test continual
- [ ] test pay
- [ ] test pay date range

Notes:
- donated and forwarded fees are now only from the executed payouts
    * this is to avoid inconsistancy in case of configuration changes
- fees from invalid payouts - not matching rules, failed execution etc. are untouched and sitting on the payout address