---
name: llm-tool-edge-rules
description: Rules for connecting `llm:agent` nodes to `tool:*` nodes in Emerald. Use when adding, moving, or validating agent tool wiring or explaining why a tool edge is invalid.
---

Use this skill when the task is specifically about agent tool wiring.

## Connect Agent Tools Correctly

- Only `llm:agent` nodes may connect to `tool:*` nodes.
- Use the agent node's `tool` source handle.
- Set `sourceHandle: "tool"` on the edge.
- Target a real `tool:*` node id.

## Keep Tool Nodes Out Of The Main Flow

- Do not place tool nodes inline on the normal execution path.
- Do not connect normal action nodes to tool nodes.
- Do not connect tool nodes to logic or return nodes with normal edges.

## Preserve Existing Tool Wiring

- Keep existing tool edge ids when editing unless replacement is required.
- When moving tool nodes visually, preserve the handle rules.
- When removing a tool node, also remove its agent tool edge.

## Recognize Invalid Examples

- `action:http` -> `tool:pipeline_run`
- `llm:prompt` -> `tool:http`
- `llm:agent` -> `tool:http` without `sourceHandle: "tool"`
- Any normal edge that targets a `tool:*` node

## Keep Agent Setups Readable

- Group tool nodes near the agent visually.
- Keep the main flow easy to scan.
- Add only the tools the agent truly needs.
