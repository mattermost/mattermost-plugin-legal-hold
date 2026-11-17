# Development stack

A Mattermost server with PostgreSQL, named after the git worktree, with the plugin built and installed.

```sh
./dev.sh up        # start the stack, create users, build and install the plugin
./dev.sh deploy    # build the plugin again and install it
./dev.sh status    # URL and logins
./dev.sh down      # stop, keep the data
./dev.sh destroy   # stop, delete the data
./dev.sh env       # environment for the Playwright tests in e2e-tests/
```

- The stack name and the port come from the worktree directory, so worktrees do not clash. `dev.sh` writes them to `.env`.
- Requirements: Docker, Go, and the Node version in `.nvmrc`.
- Logins: `admin`, `user-1`, `user-2` and `user-3`, with the password that `./dev.sh status` shows.
- The server image is `mattermost/mattermost-enterprise-edition:11.7`, the minimum server version. Set `MM_IMAGE` to test another one.
- Developer and testing modes let the plugin run without an Enterprise license. Put a license in `dev/license.txt` to test with one.
- LDAP groups need an LDAP server, which this stack does not include.
