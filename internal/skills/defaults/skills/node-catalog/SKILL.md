---
name: node-catalog
description: Compact reference for important Emerald node families, common config shapes, and when to use each category. Use when choosing nodes, understanding existing nodes, or mapping a user request to pipeline components.
---

Use this skill when the task is to identify which node types fit a pipeline step.

## Start From Node Families

- Triggers start flows:
  - `trigger:manual`
  - `trigger:cron`
  - `trigger:webhook`
  - `trigger:channel_message`
- Actions run on the normal execution path:
  - `action:http`
  - `action:shell_command`
  - `action:lua`
  - `action:pipeline_get`
  - `action:pipeline_run`
- Logic nodes shape control flow and returned data:
  - `logic:condition`
  - `logic:switch`
  - `logic:merge`
  - `logic:aggregate`
  - `logic:sort`
  - `logic:limit`
  - `logic:remove_duplicates`
  - `logic:summarize`
  - `logic:return`
- LLM nodes handle prompting and agents:
  - `llm:prompt`
  - `llm:agent`
- Tool nodes are callable helpers for `llm:agent`:
  - `tool:http`
  - `tool:shell_command`
  - `tool:pipeline_list`
  - `tool:pipeline_get`
  - `tool:pipeline_create`
  - `tool:pipeline_update`
  - `tool:pipeline_delete`
  - `tool:pipeline_run`

## Know The Most Important Execution Patterns

- Use `action:http` for direct HTTP requests in the main flow.
- Use `action:shell_command` for local shell execution in the main flow.
- Use `action:lua` for compact custom transformations when built-in transform nodes are not enough.
- Use `action:pipeline_run` to call another pipeline from the normal execution path.
- Use `logic:return` when the pipeline should produce a structured result for callers or for inspection.
- Use `llm:prompt` for a single prompt/response step.
- Use `llm:agent` when the model should choose and call tool nodes.

## Use The Right Logic Node

- Use `logic:condition` for one yes/no branch.
- Use `logic:switch` for several named branches.
- Use `logic:merge` to combine upstream objects into one object.
- Use `logic:aggregate` to collect upstream results into arrays plus metadata.
- Use `logic:sort` to sort an array at `inputPath` and write the result back to `outputPath`.
- Use `logic:limit` to keep only the first N items from an array at `inputPath`.
- Use `logic:remove_duplicates` to deduplicate an array by whole item or by one field path.
- Use `logic:summarize` to compute grouped or overall metrics such as `count`, `sum`, `avg`, `min`, and `max`.
- Use `logic:return` to stop the flow and return data.

## Remember The Transformation Palette

- In the editor, `logic:merge`, `logic:aggregate`, `logic:sort`, `logic:limit`, `logic:remove_duplicates`, and `logic:summarize` are grouped under transformation-oriented menu paths.
- `action:lua` remains a general action node and is best treated as a flexible fallback when the built-in transform nodes do not fit.
- The built-in array transform nodes clone the current payload, read an array from `inputPath`, and write the transformed result to `outputPath` so unrelated fields stay available downstream.

## Remember Infrastructure Coverage

- Infrastructure action and tool nodes also exist for Proxmox and Kubernetes.
- Prefer reading the current pipeline snapshot before assuming which infrastructure nodes are available in use.
- For broad pipeline construction examples and full JSON shapes, read `pipeline-builder`.

## Favor Clear Canvas Design

- Use explicit labels.
- Keep ids readable and stable.
- Keep the normal execution path visually separate from agent tool clusters.
- Avoid adding nodes that do not change behavior.
