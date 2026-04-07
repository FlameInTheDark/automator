---
name: pipeline-builder
description: Design, create, and safely edit Automator pipelines. Use this skill when an LLM needs to produce valid nodes/edges JSON for pipeline tools or reason about existing workflows.
---

Use this skill whenever the task is to create, inspect, update, or delete Automator pipelines.

This app stores pipelines as React Flow-style JSON. Good pipelines here are small, explicit, and easy to debug. Prefer a working first version over an overcomplicated graph.

## Primary Goals

- Produce valid pipeline definitions for `create_pipeline` and `update_pipeline`
- Preserve existing behavior when editing an existing pipeline
- Keep graphs readable
- Choose safe defaults: `draft` unless activation is clearly requested
- Make returned/output data easy for downstream nodes or users to understand

## Recommended Workflow

1. If editing an existing pipeline, inspect it first:
   - Use `list_pipelines`
   - Set `includeDefinition: true`
   - Filter by `pipelineId` or exact `pipelineName`
2. Understand the intent:
   - trigger source
   - data needed
   - decisions/branching
   - final output or side effects
3. Build the smallest pipeline that satisfies the goal.
4. Add `logic:return` when the pipeline is meant to return structured data to a caller or sub-pipeline tool.
5. Keep status as `draft` unless the user explicitly wants it active now.
6. After creating or updating, summarize what the pipeline does and call out any assumptions.

## Core Structure

A pipeline definition uses:

- `nodes`: array of node objects
- `edges`: array of edge objects
- optional `viewport`

Node shape:

```json
{
  "id": "trigger-manual-1",
  "position": { "x": 120, "y": 220 },
  "data": {
    "label": "Manual Trigger",
    "type": "trigger:manual",
    "config": {},
    "enabled": true
  }
}
```

Edge shape:

```json
{
  "id": "edge-trigger-http",
  "source": "trigger-manual-1",
  "target": "action-http-1"
}
```

Tool edge from an agent to a tool node:

```json
{
  "id": "edge-agent-tool-list",
  "source": "llm-agent-1",
  "sourceHandle": "tool",
  "target": "tool-pipeline-list-1"
}
```

This is the only valid way to connect a `tool:*` node.

## Hard Rules

- `logic:return` is limited to one per pipeline
- Trigger nodes should start flows; they do not take normal input edges
- Tool nodes are not part of the main execution chain
- `tool:*` nodes can only be used by `llm:agent`
- Tool nodes connect only from an `llm:agent` node's bottom `tool` handle
- Never connect a `tool:*` node with a normal edge
- Never use the `tool` handle to target anything except a `tool:*` node
- The backend validates these rules and rejects invalid pipeline definitions
- `logic:return` ends the pipeline and should not be treated like a normal fan-out node
- For `logic:condition`, branch handles are:
  - `true`
  - `false`
- For `logic:switch`, each condition needs a stable `config.conditions[].id`
  - edges use that id as `sourceHandle`
  - the fallback branch uses `sourceHandle: "default"`
- Preserve existing node IDs and edge IDs when editing unless there is a strong reason to replace them

## Important Node Types

Triggers:

- `trigger:manual`
- `trigger:cron`
- `trigger:webhook`
- `trigger:channel_message`

Actions:

- `action:http`
- `action:shell_command`
- `action:lua`
- `action:proxmox_list_nodes`
- `action:proxmox_list_workloads`
- `action:vm_start`
- `action:vm_stop`
- `action:vm_clone`
- `action:channel_send_message`
- `action:channel_send_and_wait`
- `action:pipeline_run`

Logic:

- `logic:condition`
- `logic:switch`
- `logic:merge`
- `logic:aggregate`
- `logic:return`

LLM:

- `llm:prompt`
- `llm:agent`

Tool nodes:

- `tool:http`
- `tool:shell_command`
- `tool:proxmox_list_nodes`
- `tool:proxmox_list_workloads`
- `tool:vm_start`
- `tool:vm_stop`
- `tool:vm_clone`
- `tool:pipeline_list`
- `tool:pipeline_create`
- `tool:pipeline_update`
- `tool:pipeline_delete`
- `tool:pipeline_run`
- `tool:channel_send_and_wait`

## Config Patterns

Manual trigger:

```json
{ "label": "Manual Trigger", "type": "trigger:manual", "config": {}, "enabled": true }
```

HTTP request:

```json
{
  "label": "HTTP Request",
  "type": "action:http",
  "config": {
    "url": "http://127.0.0.1:8080/api/v1/health",
    "method": "GET",
    "headers": {},
    "body": ""
  },
  "enabled": true
}
```

Condition:

```json
{
  "label": "Condition",
  "type": "logic:condition",
  "config": {
    "expression": "input.response.status == \"ok\""
  },
  "enabled": true
}
```

Switch:

```json
{
  "label": "Switch",
  "type": "logic:switch",
  "config": {
    "conditions": [
      {
        "id": "healthy",
        "label": "Healthy",
        "expression": "input.status == \"ok\""
      },
      {
        "id": "busy",
        "label": "Busy",
        "expression": "input.load > 0.8"
      }
    ]
  },
  "enabled": true
}
```

Return:

```json
{
  "label": "Return",
  "type": "logic:return",
  "config": {
    "value": "{{input}}"
  },
  "enabled": true
}
```

LLM prompt:

```json
{
  "label": "LLM Prompt",
  "type": "llm:prompt",
  "config": {
    "providerId": "",
    "prompt": "Context:\\n{{input.nodes}}\\n\\nSummarize the cluster.",
    "model": "",
    "temperature": 0.7,
    "max_tokens": 1024
  },
  "enabled": true
}
```

LLM agent:

```json
{
  "label": "LLM Agent",
  "type": "llm:agent",
  "config": {
    "providerId": "",
    "prompt": "User request:\\n{{input.message.content}}",
    "model": "",
    "temperature": 0.7,
    "max_tokens": 1024,
    "enableSkills": true
  },
  "enabled": true
}
```

Run pipeline action:

```json
{
  "label": "Run Pipeline",
  "type": "action:pipeline_run",
  "config": {
    "pipelineId": "TARGET_PIPELINE_ID",
    "params": "{\"request\":\"{{input.message.content}}\"}"
  },
  "enabled": true
}
```

Run pipeline tool:

```json
{
  "label": "Run Support Pipeline",
  "type": "tool:pipeline_run",
  "config": {
    "pipelineId": "TARGET_PIPELINE_ID",
    "toolName": "run_support_pipeline",
    "toolDescription": "Run the support pipeline and return its structured result.",
    "allowModelPipelineId": false
  },
  "enabled": true
}
```

## Templates vs Expressions

Template syntax is for text/config fields:

- `{{input}}`
- `{{input.nodes}}`
- `{{input.nodes[0].status}}`

Expression syntax is for `logic:condition` and `logic:switch` expressions:

- `input.status == "ok"`
- `input.response.status_code >= 200 && input.response.status_code < 300`

Do not put template braces inside Expr expressions unless the specific field is documented as a text template.

## Good Editing Practices

When updating an existing pipeline:

- inspect it first with `list_pipelines` and `includeDefinition: true`
- preserve node IDs when keeping the same node
- preserve edge IDs when keeping the same link
- only replace nodes/edges that actually need to change
- if you add a switch branch, also add the matching edge `sourceHandle`
- if you add or remove tool nodes for an agent, keep tool edges separate from normal execution edges

## Good Creation Practices

- Start with one clear trigger
- Add one action at a time
- Use `logic:return` for structured outputs
- Prefer clear labels
- Use readable IDs such as `trigger-manual-1`, `action-http-1`, `logic-return-1`
- Positions only need to be approximate and readable left-to-right
- Omit noisy UI-only fields like `selected`, `dragging`, or `measured`

## Example: Minimal Manual Pipeline

This is a strong default when the user wants a simple pipeline that fetches something and returns it:

```json
{
  "name": "Check Health",
  "description": "Fetch the health endpoint and return the response.",
  "status": "draft",
  "nodes": [
    {
      "id": "trigger-manual-1",
      "position": { "x": 120, "y": 220 },
      "data": {
        "label": "Manual Trigger",
        "type": "trigger:manual",
        "config": {},
        "enabled": true
      }
    },
    {
      "id": "action-http-1",
      "position": { "x": 380, "y": 220 },
      "data": {
        "label": "HTTP Request",
        "type": "action:http",
        "config": {
          "url": "http://127.0.0.1:8080/api/v1/health",
          "method": "GET",
          "headers": {},
          "body": ""
        },
        "enabled": true
      }
    },
    {
      "id": "logic-return-1",
      "position": { "x": 650, "y": 220 },
      "data": {
        "label": "Return",
        "type": "logic:return",
        "config": {
          "value": "{{input}}"
        },
        "enabled": true
      }
    }
  ],
  "edges": [
    {
      "id": "edge-trigger-http",
      "source": "trigger-manual-1",
      "target": "action-http-1"
    },
    {
      "id": "edge-http-return",
      "source": "action-http-1",
      "target": "logic-return-1"
    }
  ],
  "viewport": {
    "x": 0,
    "y": 0,
    "zoom": 1
  }
}
```

## Example: Agent With Tools

When building an agent pipeline:

- put the main execution path on the canvas
- connect tool nodes from the agent node's `tool` handle
- do not place tool nodes inline on the normal action chain

Typical flow:

1. `trigger:channel_message`
2. `llm:agent`
3. `action:channel_send_message`

Connected tools:

- `tool:pipeline_list`
- `tool:pipeline_run`
- `tool:pipeline_create`
- `tool:pipeline_update`

## Decision Heuristics

Use:

- `logic:condition` for one yes/no split
- `logic:switch` for multiple named branches
- `logic:merge` when you want to merge upstream objects
- `logic:aggregate` when you want collected upstream arrays and metadata
- `logic:return` when the pipeline is used as a callable unit

## When Unsure

- prefer `draft`
- prefer a manual trigger
- prefer a `Return` node for inspectable outputs
- prefer fewer nodes
- prefer explicit labels and explicit JSON config
- inspect an existing pipeline before editing it

When the user asks for a pipeline, return or submit a complete, valid structure rather than vague pseudocode.
