# Sample Data

This directory contains `qqwry-2021.04.14.dat` as a convenience sample for local testing, examples, and CLI demos.

Important notes:

- The sample is dated `2021-04-14` and may be outdated.
- Do not treat it as authoritative production data.
- For production use, provide your own current QQWry-compatible `.dat` file.
- QQWry / 纯真 IP data rights belong to their respective owners.
- This project focuses on parsing and querying compatible `.dat` files, not distributing fresh IP-location data.

Example:

```shell
go run ./cmd/iploc --db data/qqwry-2021.04.14.dat 127.0.0.1
```
