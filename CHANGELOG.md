# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/).

## [1.0.0] — 2026-10-14

### Added
- Direct message from a *Reactions* bot when someone reacts to your message, with who, which
  emoji, where, a quote and a link.
- Reactions within a minute are grouped into one message; reactions taken back within that
  minute are dropped.
- No notifications for your own reactions, bots, system messages, or channels you've muted or
  left.
- **Settings → Reaction Notifications**: all messages, only direct and group messages, or off.
- Cluster-safe sender: each group is claimed atomically before it is sent.
- One build for Mattermost 10.11+, verified on 11.7 and 12.0.
