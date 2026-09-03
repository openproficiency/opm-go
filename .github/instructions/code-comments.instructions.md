---
description: Instructions for organizing and commenting code
applyTo: "**/*.go"
---

# Code Comment Instructions

Use comments to make files and function logic easy to scan.
Do not narrate self-explanatory code.

## File Organization

- Group functions by theme and alphabetize them within each group.
- Add section headings only when a file has more than one section.
- Example order. Don't use litterally 
  1. Imports
  2. Constants
  3. Public
    - types
    - classes
    - functions
    - etc.
  6. Private
    - types
    - classes
    - functions
    - etc.

Import headings use one line:

```ts
// Built-in Imports
import ...
// Installed Imports
import ...
// Project Imports
import ...
```

All other section headings use three lines:

```ts
//
// PUBLIC {group name}
//
```

## Functions, Types, Classes, etc.

- Keep function descriptions to one line.
- Use one short comment to state the purpose of the following logical block.
- Write block comments as pseudo-code so they outline the function when read
  alone.
