# Micro/Actions

Example usage:

```
name: M3O

on: 
  push:
  pull_request:
    branches:
      - master

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Check out repository
        uses: actions/checkout@v2
      - name: Build Services
        uses: micro/actions@1.9.22
        with:
          CLIENT_ID: ${{ secrets.M3O_CLIENT_ID }}
          CLIENT_SECRET: ${{ secrets.M3O_CLIENT_SECRET }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```