# Templates

Emerald uses `{{ ... }}` templates in string-based config fields such as prompts, URLs, headers, request bodies, messages, and `action:pipeline_run` params.

## Runtime Sources

- `input` is the full current payload for the node.
- Top-level input keys are also exposed directly, so `{{status_code}}` and `{{input.status_code}}` can both work when that key exists.
- `secret` exposes runtime secrets such as `{{secret.api_token}}`.
- `$('node-id')` exposes the decoded output of a specific node that has already completed in the current run.

Examples:

```text
{{input.response.status_code}}
{{input.items[0].name}}
{{secret.api_token}}
{{$('action:http-1775583878229').response.status_code}}
```

## Cross-Node Lookups

Use `{{$('node-id').path.to.value}}` when you need a stable reference to one earlier node output instead of following the current payload shape.

This is especially useful when:

- logic nodes wrapped earlier payloads under their own `input` key
- several upstream branches produced similarly named fields
- a later node needs one exact upstream result regardless of intermediate reshaping

Rules:

- the referenced node ID must exist
- that node must already have executed in the current run
- the requested path must exist on that node output

If any of those checks fail, template rendering returns an error instead of silently producing an empty value.

## Rendering Behavior

- Dot syntax reads nested objects, such as `{{input.response.status}}`.
- Zero-based indexes read arrays, such as `{{input.items[0]}}`.
- Objects and arrays render as JSON strings when inserted into text fields.
- Templating happens before node execution code sees config values.

## Common Patterns

Render the whole payload:

```text
{{input}}
```

Mix current input, secret values, and an earlier node result:

```json
{"status":"{{$('action:http-1775583878229').response.status_code}}","message":"{{input.message.content}}","token":"{{secret.api_token}}"}
```

Pass shaped params into a called pipeline:

```json
{"request":"{{input.message.content}}","httpStatus":"{{$('action-http-1').response.status_code}}"}
```

If `action:pipeline_run` `config.params` is empty, Emerald passes the current input object through unchanged.

## Autocomplete

The editor suggests template fields from:

- the current payload
- configured secrets
- sampled outputs from previously executed nodes, grouped by source node

Cross-node suggestions insert `{{$('node-id').path}}` snippets and only resolve after that node has run in the current execution path.

## Templates vs Expressions

- Use templates in string-based config fields.
- Use Expr syntax in `logic:condition` and `logic:switch`.

Examples:

```text
{{input.status_code}}
input.status_code == 200
```

For Expr-specific guidance, see the [expression guide](../expressions/README.md).
