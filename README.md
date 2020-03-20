# Actions

Actions is a GitHub Action for Micro services.

# Overview 

Assuming a mono-repo folder structure, where the top level directories are services,
this GitHub action extracts the services which were modified in the last commit. It
works for both single commits and pull requests.

The GitHub action requires one input: githubToken, this is provided by default by
GitHub actions, and can be accessed via: `${{ secrets.GITHUB_TOKEN }}`.

The outputs provided are lists of services which have been changed, encoded in a string
with ` ` as the seperator. It can be used as an array in bash as demonstrated below.

## How to Use
```
name: extract-services

on:
  push:
    
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Check out repository
        uses: actions/checkout@v2
      - name: Detect which services changed
        id: services_changed
        uses: micro/actions@1.0.19
        with:
          githubToken: ${{ secrets.GITHUB_TOKEN }}
      - name: test
        run: |
          cat $HOME/changes.json
          echo '${{ steps.services_changed.outputs.services }}'

          echo Logging services added
          services=(${{ steps.services_changed.outputs.services_added }})
          for dir in "${services[@]}"; do
            echo Added $dir
          done

          echo Logging services modified
          services=(${{ steps.services_changed.outputs.services_modified }})
          for dir in "${services[@]}"; do
            echo Modified $dir
          done

          echo Logging services removed
          services=(${{ steps.services_changed.outputs.services_removed }})
          for dir in "${services[@]}"; do
            echo Removed $dir
          done

          echo Done
```
