# Troubleshooting

## `no BuildKit endpoint provided and BUILDKIT_HOST not set`

`ch` doesn't know where BuildKit runs. Either start a local daemon and export the address:

```bash
eval $(ch dev buildkitd start)
```

or set `BUILDKIT_HOST`, or `buildkit.address` in [`hive.yml`](configuration/hive.md#buildkit).

## `failed to connect to BuildKit at ...`

- `buildkit.address` in `hive.yml` **overrides** `$BUILDKIT_HOST`. Remove it if you want the environment variable to win.
- Check the daemon is reachable with `ch wait --buildkitd --timeout 30s`.
- `ch dev buildkitd status` and `ch dev buildkitd logs` show the state of the local daemon.

## `dist/ not found — run 'ch generate' first`

`build`, `test` and `sbom` work on the output of `ch generate`. Run it first, or pass `--generate`.

## An image is built before its parent / a parent isn't found

Check the `FROM __hive__/<image>` name matches the parent's directory name, or declare the relationship with [`depends_on`](configuration/image.md#depends_on). See [How dependency resolution works](how-it-works/dependency-resolution.md).

## More logs

Run with `--log-level debug` (or `LOG_LEVEL=debug`). Still stuck? Open an issue at the [issue tracker](https://github.com/ContainerHive/ContainerHive/issues).
