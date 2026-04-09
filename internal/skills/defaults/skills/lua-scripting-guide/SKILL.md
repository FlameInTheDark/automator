---
name: lua-scripting-guide
description: Guide for `action:lua` nodes, including how runtime data is exposed, how `input` works, Lua table conversion, and how return values become downstream output. Use when creating or explaining Lua nodes.
---

Use this skill when the task is to create, review, or explain an `action:lua` node.

## Build The Node Correctly

- Use `action:lua` on the normal execution chain.
- Put the script in `data.config.script`.
- Do not wrap the script in a helper function unless the script itself needs one internally.
- There is no `tool:lua` node today.

## Read Runtime Data The Right Way

- The script is executed directly.
- The current node input is exposed as a global named `input`.
- Prefer `input.<field>` as the source of truth.
- Top-level input keys are also mirrored as individual Lua globals, but `input` is the clearest form.

## Respect Data Conversion Rules

- Objects and maps become Lua tables with string keys.
- Arrays and slices become 1-based Lua arrays.
- Template indexes are zero-based, but Lua indexes are 1-based.
- Example:
  - template: `{{input.items[0]}}`
  - Lua: `input.items[1]`

## Return Output Correctly

- Use top-level `return` to produce node output.
- Return a table with named keys for structured output.
- Returning a primitive becomes an object shaped like `{"result": <value>}` downstream.
- Returning `nil` produces an empty output object.
- Contiguous numeric tables become JSON arrays.
- Mixed numeric and string keys are treated as an object.

## Prefer Practical Patterns

```lua
local first = input.items and input.items[1] or nil

return {
  ok = input.status == "ready",
  first_item = first,
  count = input.items and #input.items or 0
}
```

## Avoid Common Mistakes

- Do not assume `input` is passed as a function argument.
- Do not use zero-based array indexes in Lua.
- Do not rely on mirrored globals when `input.<field>` is clearer.
- Do not forget that the script can return structured tables directly.
