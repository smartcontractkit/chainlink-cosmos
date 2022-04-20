# Withdrawing Payments from Feeds

When Node Operators report data to a feed, they are compensated with LINK thats owned by the feed. Node operators can then withdraw the funds owed to them via gauntlet using this section below

## Sample Command Usage

Make sure to set up gauntlet using the steps described in [Getting Started With Gauntlet](./GettingStartedWithGauntlet.md) before attempting to run the following command

```
yarn gauntlet ocr2:withdraw_payment --network=mainnet --transmitter=<NODE_OPERATOR_ADDRESS> <FEED_CONTRACT_ADDRESS>
```

