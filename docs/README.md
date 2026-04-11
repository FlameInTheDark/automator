# Documentation

This directory collects the project documentation that goes deeper than the root README.

## Categories

- [Nodes](./nodes/README.md) - built-in node families, execution behavior, and payload-shape notes.
- [Templates](./templates/README.md) - how `{{...}}` interpolation works for `input`, `secret`, and executed-node lookups.
- [Expressions](./expressions/README.md) - short introduction to Emerald's expression language support and links to the official Expr docs.
- [Settings](./settings/README.md) - app settings navigation, section IDs, and deep-link behavior.
- [Plugins](./plugins/README.md) - plugin reference for manifests, SDK types, runtime behavior, and troubleshooting.
- [Plugin Tutorial](./plugins/tutorial.md) - step-by-step walkthrough for building and loading a custom plugin node.

## Suggested Reading Order

If you are new to Emerald, this order works well:

1. Read the [node reference](./nodes/README.md) to understand how pipelines are composed.
2. Read the [template guide](./templates/README.md) before wiring prompts, params, headers, or cross-node lookups.
3. Read the [expression guide](./expressions/README.md) before writing `logic:condition` or `logic:switch` rules.
4. Read the [settings guide](./settings/README.md) when configuring channels, secrets, AI providers, plugin bundles, or users.
5. Read the [plugin tutorial](./plugins/tutorial.md) if you want to build your first plugin.
6. Keep the [plugin reference](./plugins/README.md) nearby while polishing manifests, fields, outputs, and troubleshooting.
