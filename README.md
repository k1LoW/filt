# filt [![Build Status](https://github.com/k1LoW/filt/workflows/build/badge.svg)](https://github.com/k1LoW/filt/actions) [![GitHub release](https://img.shields.io/github/release/k1LoW/filt.svg)](https://github.com/k1LoW/filt/releases)

`filt` is a interactive/realtime stream filter ( also known as _"trial-and-error pipe"_ ).

![screencast](doc/screencast.svg)

## Usage

``` console
$ tail -F /var/log/nginx/access.log | filt
```

and enter `Ctrl+C`.

### How to filter files by trial and error

You can use `--buffered` ( `-b` ) option

``` console
$ cat /var/log/nginx/access.log | filt -b
```

and enter `Ctrl+C`.

### How to exit from filt prompt

Input "exit" to prompt or enter `Ctrl+C`.

### Enable or Disable saving history

**Enable:**

``` console
$ filt config history.enable true
```

**Disable:**

``` console
$ filt config history.enable false
```

## Install

**homebrew tap:**

```console
$ brew install k1LoW/tap/filt
```

**manually:**

Download binany from [releases page](https://github.com/k1LoW/filt/releases)

**go get:**

```console
$ go get github.com/k1LoW/filt
```

## Alternatives

- [up](https://github.com/akavel/up): up is the Ultimate Plumber, a tool for writing Linux pipes in a terminal-based UI interactively, with instant live preview of command results.
