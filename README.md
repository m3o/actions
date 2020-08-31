# M3O GitHub Actions

GitHub Actions for M3O Services

## Overview

Soon...

## Usage

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
        uses: m3o/action@1.9.22
        with:
          CLIENT_ID: ${{ secrets.M3O_CLIENT_ID }}
          CLIENT_SECRET: ${{ secrets.M3O_CLIENT_SECRET }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```
