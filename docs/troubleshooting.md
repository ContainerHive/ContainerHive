# Troubleshooting

## `no BuildKit endpoint provided and BUILDKIT_HOST not set`

`ch` doesn't know where BuildKit runs. Either start a local daemon and export the address:

```bash
eval $(ch dev buildkitd start)
```

or set `BUILDKIT_HOST`, or `buildkit.address` in [`hive.yml`](configuration/hive.md#buildkit).

## `failed to connect to BuildKit at ...`

- `buildkit.address` in `hive.yml` **overrides** `$BUILDKIT_HOST`. Remove it if you want the environment variable to win.
- Check the daemon is reachable with `ch wait --buildkitd --timeout 30s`. `wait` reads only `$BUILDKIT_HOST` and ignores `buildkit.address`, so if you configure the address in `hive.yml`, export it first: `BUILDKIT_HOST=<your buildkit.address> ch wait --buildkitd`.
- `ch dev buildkitd status` and `ch dev buildkitd logs` show the state of the local daemon.

## `dist/ not found — run 'ch generate' first`

`build`, `test` and `sbom` work on the output of `ch generate`. Run it first, or pass `--generate`.

## An image is built before its parent / a parent isn't found

Check the `FROM __hive__/<image>:<tag>` reference names the parent's directory and one of its tags, or declare the relationship with [`depends_on`](configuration/image.md#depends_on). See [How dependency resolution works](how-it-works/dependency-resolution.md).

## The build fails with a BuildKit connection error

Transient connection failures (`broken pipe`, `connection reset by peer`, `Unavailable`, worker listing timeouts) are retried up to three times on a fresh connection. Each retry waits 2, 5, then 10 seconds. Genuine build failures are never retried. If all retries fail, check that the BuildKit daemon is alive; for a remote daemon, check the network path and the proxy or load balancer timeouts. See [BuildKit setup](configuration/hive.md#buildkit).

## More logs

Run with `--log-level debug` (or `LOG_LEVEL=debug`). Still stuck? Open an issue at the [issue tracker](https://github.com/ContainerHive/ContainerHive/issues).
