# @zeb-link/zeb

Zeb, the command-line client for [Zebra Link](https://zeblink.io). Create and
manage short links, collections, domains, and spaces from the terminal, or from
a script.

This package ships a **prebuilt native binary** (Go) for your platform. Node is
used to install it, not to run it. The binary lives in a per-platform package
that npm resolves automatically; the Go source is not included.

## Install

```bash
npm i -g @zeb-link/zeb      # or: pnpm add -g @zeb-link/zeb
zeb login
```

`zeb login` prompts for your Zebra Link API key and validates it. That is the
only setup step.

## Quick start

```bash
zeb https://example.com      # create a short link
zeb links                    # list links
zeb tui                      # interactive browser
zeb --help
```

Every command takes `--json` for machine-readable output.

## Supported platforms

macOS, Linux, and Windows on x64 or arm64.

Full documentation and source:
**https://github.com/zeb-link/zeb**

Bug reports and ideas are welcome — open an issue or email <support@zeblink.io>.
