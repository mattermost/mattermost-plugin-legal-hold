Start Mattermost server:

- `cd <path>/mattermost/server`
- `make test-data`
- `make run-server`

Build and deploy plugin

- `make deploy`

Run test:

- `cd <path>/mattermost-plugin-legal-hold/e2e-tests`
- `npm run test` to run in multiple projects such as `chrome`, `firefox` and `ipad`.
- `npm run test -- --project=chrome` to run in specific project such as `chrome`.

To see the test report:

- `cd <path>/mattermost-plugin-legal-hold/e2e-tests`
- `npm run show-report`
- Navigate to http://localhost:8065

To see test screenshots:

- `cd <path>/mattermost-plugin-legal-hold/e2e-tests/screenshots`
