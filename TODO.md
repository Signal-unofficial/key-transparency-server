# TODOs

- Test with an auditor(s) (see: [#1](https://github.com/Signal-unofficial/key-transparency-server/issues/1))
- Add PowerShell Maven wrapper to [`filter-key-updates`](./filter-key-updates/)
- Optimize Docker build stages by building every module in one stage and copying from it.
  The alternative would require a way to install all the non-local Go dependencies.
- Migrate from using AWS access and secret keys in environment variables
  to using Docker secrets, either through the AWS secrets manager or custom images.
