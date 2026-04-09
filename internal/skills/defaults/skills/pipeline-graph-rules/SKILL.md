---
name: pipeline-graph-rules
description: Reference for valid Automator node and edge topology, branching handles, and live-edit safety rules. Use when reasoning about graph structure, adding or removing nodes, reconnecting edges, or validating pipeline edits.
---

Use this skill when the task is about whether a pipeline graph is structurally valid.

## Validate The Snapshot Shape

- Treat pipelines as React Flow style JSON with `nodes`, `edges`, and optional `viewport`.
- Treat the browser-provided snapshot as the source of truth for live editor work, even if it is not saved yet.
- Preserve existing node ids and edge ids unless a replacement is truly required.
- Keep positions readable and left-to-right; they are UI state, not execution logic.

## Enforce Core Topology Rules

- Allow only one `logic:return` node per pipeline.
- Treat trigger nodes as flow entrypoints. Do not give them normal incoming edges.
- Treat `visual:group` nodes as layout-only. Do not connect edges to them.
- Do not give `logic:return` any outgoing edge.
- Keep tool nodes out of the main execution chain.

## Enforce LLM Tool Wiring

- Connect `tool:*` nodes only from an `llm:agent` node.
- Use `sourceHandle: "tool"` for agent-to-tool edges.
- Never connect a `tool:*` node with a normal edge.
- Do not target a non-tool node from the `tool` handle.

## Respect Branch Handle Rules

- For `logic:condition`, use `sourceHandle: "true"` and `sourceHandle: "false"` for the two branches.
- For `logic:switch`, each branch condition needs a stable `config.conditions[].id`.
- Use the condition id as the edge `sourceHandle`.
- Use `sourceHandle: "default"` for the fallback branch.

## Edit Conservatively

- Prefer the smallest operation set that satisfies the request.
- Preserve unrelated nodes, edges, labels, and config fields.
- When updating a node or edge, keep the existing object shape intact except for the requested change.
- When deleting nodes, also remove or replace any incident edges so the graph stays valid.

## Common Invalid States

- Edge points to a missing node id.
- Multiple `logic:return` nodes exist.
- A trigger has a normal incoming edge.
- A tool node is inserted into the normal execution chain.
- A switch branch edge uses a label instead of `config.conditions[].id`.

## Good Defaults

- Start simple.
- Use one clear trigger.
- Add one action at a time.
- Use `logic:return` when the pipeline should return structured output.
