# App Settings

Emerald app settings now use a left-side navigation rail instead of the older top tab strip. The active section is driven by the `section` query parameter, which makes settings views deep-linkable.

## Section Groups

### Infrastructure

- `proxmox`
- `kubernetes`

### Messaging

- `channels`

### AI

- `ai.providers`
- `ai.assistants`

### Security

- `secrets`
- `users`

### Extensibility

- `plugins`

## Deep Links

You can open settings directly to a section with URLs like:

```text
/settings?section=channels
/settings?section=ai.assistants
/settings?section=plugins
```

If `section` is missing or unknown, Emerald falls back to the default settings section.

## Notes

- The same section IDs drive the desktop layout and the mobile selector fallback.
- Plugin bundle health and rediscovery now live in `Plugins`, not under `Secrets`.
- Shared assistant prompt profiles live in `AI -> Assistants`.
- Provider configuration lives in `AI -> Providers`.
- User management and default-password changes live in `Security -> Users`.
