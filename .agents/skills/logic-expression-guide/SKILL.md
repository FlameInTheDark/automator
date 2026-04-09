---
name: logic-expression-guide
description: Guide for expr-based condition and switch logic in Emerald pipelines. Use when writing or debugging `logic:condition` and `logic:switch` expressions or explaining what variables are available there.
---

Use this skill when the task is about expressions in logic nodes.

## Use Expr Semantics

- `logic:condition` and expression-based `logic:switch` branches use expr-lang style expressions.
- The expression must evaluate to a boolean.
- `input` is available as the full payload.
- Top-level input keys are also exposed directly.

## Write Typical Expressions

```text
input.status == "ready"
retries > 3
input.response.status_code >= 200 && input.response.status_code < 300
input.cluster == "prod" && input.enabled == true
```

## Keep Expressions Different From Templates

- Use plain expr syntax in condition fields.
- Do not wrap the whole expression in `{{ ... }}`.
- Templating runs before expression evaluation only when the field supports templating, but the normal condition pattern is plain expr syntax.

## Prefer Clear Checks

- Compare explicit fields instead of serializing whole objects.
- Use `input.<field>` when the source should stay obvious.
- Keep branch logic short and readable.

## Avoid Common Mistakes

- Do not return a string or number when a boolean is required.
- Do not use Lua syntax inside expr fields.
- Do not mix template braces into a normal condition expression.
