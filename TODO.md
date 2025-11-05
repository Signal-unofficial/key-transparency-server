# TODOs

- Replace `account-db: mock` config with a real DynamoDB table
- Test with an auditor(s)
- Add PowerShell Maven wrapper to [`filter-key-updates`](./filter-key-updates/)
- Optimize Docker build stages
- Add built Docker entrypoints to PATH:

  ```Dockerfile
  ENV PATH=${PATH}:/abs/path/to/script
  ```
