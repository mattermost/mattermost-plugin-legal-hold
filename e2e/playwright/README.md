In order to get your environment set up to run [Playwright](https://playwright.dev) tests, you can run `./setup-environment`, or run equivalent commands for your current setup.

What this script does:

-   Navigate to the folder above `mattermost-plugin-legal-hold`
-   Clone `mattermost` (if it is already cloned there, please have a clean git index to avoid issues with conflicts)
-   `cd mattermost`
-   Install webapp dependencies - `cd webapp && npm i`
-   Install Playwright test dependencies - `cd ../e2e-tests/playwright && npm i`
-   Install Playwright - `npx install playwright`
-   Install Legal Hold plugin e2e dependencies - `cd ../../../mattermost-plugin-legal-hold/e2e/playwright && npm i`
-   Build and deploy plugin with e2e support - `make deploy`

---

Then to run the tests:

Start Mattermost server:

-   `cd <path>/mattermost/server`
-   `make test-data`
-   `make run-server`

Run test:

-   `cd <path>/mattermost-plugin-legal-hold/e2e/playwright`
-   `npm run test` to run in multiple projects such as `chrome`, `firefox` and `ipad`.
-   `npm run test -- --project=chrome` to run in specific project such as `chrome`.

To see the test report:

-   `cd <path>/mattermost-plugin-legal-hold/e2e/playwright`
-   `npm run show-report`
-   Navigate to http://localhost:8065

To see test screenshots:

-   `cd <path>/mattermost-plugin-legal-hold/e2e/playwright/screenshots`
