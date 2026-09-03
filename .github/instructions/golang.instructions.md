---
description: Instructions for developing Go code
applyTo: '*.go'
---

- Keep code simple and flat. Do not add an abstraction until more than one
  caller needs it.
- Prefer the options with the least code.
- Prefer trusted well-known packages over of custom implementations.
- Prefer readability and maintainability.
- Prefer asynchronous processing.

## Model Boundary

Treat the Open Proficiency Model `v0.1` schemas as the source of truth.

If there are any doubts, check the docs, specifications, and schema int the Open Proficiency Model repo https://github.com/openproficiency/model.

### Variables

Do not use generic names for variables, functions, structs, etc.
Bad examples: nk, n, myValue, newValue

Do not use generic prefixes.
Bad example: "newClient".

When filling out structs, put one field plus value per line.
Example:

```go
MyStruct{
    Field1: "value 1",
    Field2: "value 2",
}
```

### Docs

Function and test descriptions dont start with the function name. Keep concise. No fluff. Ideally 2 lines max.
Function descriptions explain basic normal usange, not how it works, not detailed usage.

### Inline Comments

Comments serve as chapter markers to help a navigate the code. It should read like psuedo code.
Insert inside methods to identify logical chunks
Insert comments to group related methods.

