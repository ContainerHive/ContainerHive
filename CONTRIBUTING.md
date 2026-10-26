# Contributing to ContainerHive

## Maintainers wanted

This project currently has a sole maintainer: Timo Reymann. To make reviews, releases, and urgent fixes independent of
one person's availability, additional maintainers are welcome. Responsibilities are small and shared, and everything is
discussed before you commit to anything. Reach out to <oss@timo-reymann.de> to talk details.

We love your input! We want to make contributing to this project as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## We develop with Github
We use github to host code, to track issues and feature requests, as well as accept pull requests.

## We Use [CircleCI](https://circleci.com/product/), So All Code Changes Happen Through Pull Requests
Pull requests are the best way to propose changes to the codebase (we use [CircleCI](https://circleci.com/product/)). We actively welcome your pull requests:

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the tests pass.
5. Make sure your code and commit lints.
6. Issue that pull request!

## Any contributions you make will be under the AGPL-3.0
In short, when you submit code changes, your submissions are understood to be under the same [AGPL-3.0](https://github.com/ContainerHive/ContainerHive/blob/main/LICENSE) that covers the project. Feel free to contact the maintainers if that's a concern.

## Report bugs using Github's issues
We use GitHub issues to track public bugs. Report a bug by opening a new issue, it's that easy!

## Write bug reports with detail, background, and sample code

**Great Bug Reports** tend to have:

- A quick summary and/or background
- Steps to reproduce
    - Be specific!
    - Give sample code if you can.
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

People *love* thorough bug reports. We're not even kidding.

To make your life easier there is also a handy template available so feel free to use it.

> ATTENTION: If you DONT provide steps to reproduce the ticket will be closed WITHOUT FURTHER NOTICE!

## Keeping upstream synced inside a company network

Use `./hack/release-snapshot.sh` to publish a snapshot image from any git ref or pull request to a registry of your
choice — for example your company registry. See the [snapshot release guide](./docs/development/snapshot-release.md).

## License
By contributing, you agree that your contributions will be licensed under its GPL-3.0.



## AI-Generated Contributions Policy

We accept AI-generated contributions to our projects, but with the following important conditions:

1. **Understanding Requirement**: The human user authoring the change must have a clear conceptual understanding of what the code does and how it works.

2. **Ownership Awareness**: The human contributor must understand that they own the intellectual property rights to the result, not the AI agent or tool that assisted them.

3. **Explicit Attribution**: The human contributor must clearly state their understanding and compliance with the above conditions through one of these methods:
   - Using "Co-Authored-By" attribution in the commit message
   - Explicitly mentioning in the GitHub pull request description that AI assistance was used and that they understand and take responsibility for the changes

4. **Right to Decline**: If these conditions are not clearly met in the contribution, we reserve the right to decline the pull request without further explanation.

5. **Quality Standards**: All contributions, whether AI-assisted or not, must meet the same quality standards and will be evaluated on their technical merit.

By contributing to this project, you agree to these terms and acknowledge that you are responsible for the code you submit, regardless of how it was generated.
