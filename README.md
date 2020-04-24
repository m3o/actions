# Micro/Actions

Example usage:

```
name: M3O

on: 
  push:
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
          API_KEY: ${{ secrets.M3O_TOKEN }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```
