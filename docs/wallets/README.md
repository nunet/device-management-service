# Cardano Eternl Wallet Integration

## Prerequisites

- Cardano Eternl Wallet on browser
- `eternl-cli` installed from https://gitlab.com/nunet/dev-tools/eternl-cli

## Generating DID from Eternl

```bash
./dms key did eternl:myaccount
```
result will be:

```
did:key:zE4MrBkkgDczQPwp789GPUJo26cxqtWSz8J47jQgakp7RnJKH
```

## Generate Capabilities

Generating new capabilities:

```bash
./dms cap new eternl:myaccount
```
This will open the browser and ask you to sign a payload in order for dms to get a did from eternl signature

Will create `myaccount.cap` file inside `cap` directory

## Granting with eternl 

```bash
./dms cap grant --cap /public --context eternl:myaccount did:key:z6MkngPHCGesTFB9SfKukYqGaDkp2fJUGiCcXKE9bg49ddfB --expiry 2030-01-02
```

This command will open the broswer, and ask you first to sign a generic message which is for dms to load the current did from eternl signature, then it will open a second browser to sign the actual capability context to grant.

This will create a capability token with cardano signature format that can be consumed.