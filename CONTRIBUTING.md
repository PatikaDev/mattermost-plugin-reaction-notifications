# Contributing

Thanks for helping! Issues and pull requests are welcome.

## Before you open a pull request

```bash
go test -race ./server/...
cd webapp && npm ci && npm run lint && npm run check-types && npm test
cd .. && make dist
cd e2e && npm ci && MM_VERSION=11.7.11 ./scripts/start-server.sh && npx playwright test
```

CI runs the same checks plus `golangci-lint` on `./server/...`, and the end-to-end suite on
every supported Mattermost line.

## Server rules

- The logic lives in `server/reactions` behind small interfaces (`Store`, `Mattermost`, `Clock`)
  so it can be unit-tested with fakes; `server/` only wires it to Mattermost.
- Every change to a pending group goes through `Service.update` (compare-and-set with retries),
  and every send goes through `Service.Flush`, which claims the group by deleting it atomically.
- Keep the notification quiet: when in doubt, don't send.

## Webapp compatibility rules

The plugin uses the **server's** React at runtime — React 17 on Mattermost 10.11, 18 on 11.x,
19 on 12.x — so one build has to work on all of them:

- Keep `@babel/preset-react` on the **classic** JSX runtime. The automatic runtime needs
  `window.ReactJSXRuntime`, which only Mattermost 12 provides.
- Use function components and hooks available since React 16.8. Avoid `defaultProps` on
  function components, string refs, legacy context, `findDOMNode`, `ReactDOM.render`,
  `useId` and `useSyncExternalStore`.
- Don't import `prop-types`, `react-bootstrap` or `react-router-dom`.
