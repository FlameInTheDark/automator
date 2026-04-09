---
name: templating-guide
description: Guide for Automator template interpolation, runtime context lookup, and pipeline parameter passing. Use when writing prompts, config fields, JSON params, or explaining how `input`, `arguments`, and other template values resolve.
---

Use this skill when the task is about `{{ ... }}` templates in node configuration.

## Use Template Syntax Correctly

- Write templates as `{{path.to.value}}`.
- Treat `input` as the full current payload.
- Access nested objects with dot syntax such as `{{input.response.status}}`.
- Access arrays with zero-based indexes such as `{{input.items[0]}}`.
- Expect rendered objects and arrays to become JSON strings when inserted into text fields.

## Know The Runtime Context

- `input` is always available.
- Top-level input keys are also exposed directly, so `{{status}}` may work when `input.status` exists.
- Prefer `{{input...}}` when clarity matters.
- Missing paths cause template rendering errors rather than quietly returning partial data.

## Understand Pipeline Parameter Passing

- `action:pipeline_run` uses `config.params` as a templated string field.
- After templating, `config.params` must be valid JSON that decodes to an object.
- If `action:pipeline_run` `config.params` is empty, the current input map is passed through to the called pipeline.
- `tool:pipeline_run` passes the called pipeline input as a normal object, not as a raw JSON string.
- Custom `tool:pipeline_run` arguments are exposed inside the called pipeline under `arguments.<name>`.

## Use Common Patterns

- Pass the whole payload through:

```text
leave action:pipeline_run config.params empty
```

- Pass a shaped object:

```json
{"request":"{{input.message.content}}","ticket":"{{arguments.ticket}}"}
```

- Return the current payload:

```text
{{input}}
```

## Keep Templates Separate From Expr Logic

- Use template syntax for string/config fields.
- Use expr syntax in `logic:condition` and `logic:switch` expressions.
- Do not wrap expr logic in `{{ ... }}` unless the field explicitly expects a text template.
