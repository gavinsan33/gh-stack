# gh-stack (fork)

A fork of [github/gh-stack](https://github.com/github/gh-stack) with fixes for rebasing stacks whose parents were rewritten. For usage and commands, see the [upstream README](https://github.com/github/gh-stack#readme) and [docs](https://github.github.com/gh-stack/).

## What's different

- **Already-applied commits are dropped on rebase.** If a branch was stacked on an older copy of its parent, conflicts on commits the new base already has (same patch, or same author, author date and subject) are resolved by keeping the base's side. Only conflicts in the branch's own commits stop the rebase.
- **Rebase base recovery.** When the old parent commit is gone, the rebase base is recovered from the branch's patch-equivalent copy of it.
- **`modify` after `sync --prune`.** Merged branches that `sync --prune` deleted are skipped when snapshotting the stack.

## Install

Requires [`gh`](https://cli.github.com/) and Git 2.36+.

```sh
gh extension remove stack 2>/dev/null   # if upstream is installed
gh extension install gavinsan33/gh-stack
```

Upgrade with `gh extension upgrade stack`. To switch back to upstream, remove it and run `gh extension install github/gh-stack`.

The update notifier only runs for upstream installs, so it won't prompt you to switch to upstream releases.

## Development

```sh
make install   # build and link this checkout as the gh stack extension
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for tests.

## License

MIT. See [LICENSE](LICENSE).
