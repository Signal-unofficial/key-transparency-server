# TODOs

- Test with an auditor(s)
- Add PowerShell Maven wrapper to [`filter-key-updates`](./filter-key-updates/)
- Optimize Docker build stages
- Add built Docker entrypoints to PATH:

  ```Dockerfile
  ENV PATH=${PATH}:/abs/path/to/script
  ```

- Connect Prometheus metrics container to `server:8083`
- Migrate from using AWS access and secret keys in environment variables
  to using Docker secrets, possibly through the AWS secrets manager.
