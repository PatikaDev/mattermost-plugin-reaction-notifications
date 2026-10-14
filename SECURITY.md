# Security policy

## Reporting a vulnerability

Please **do not open a public issue** for security problems. Use GitHub's
[private vulnerability reporting](https://github.com/PatikaDev/mattermost-plugin-reaction-notifications/security/advisories/new)
for this repository instead. We aim to acknowledge reports within 5 working days.

## Scope

The plugin only reads what it needs to describe a reaction (the post, the users involved and
the channel) and only ever sends that description to the post's author, who can already see
the post and its reactions. It re-checks right before sending that the author is still a
member of the channel and hasn't muted it or turned notifications off. It has no HTTP API.
Pending reactions are stored for about a minute in the plugin's key-value store.

## Supported versions

Security fixes are released for the latest minor version.
