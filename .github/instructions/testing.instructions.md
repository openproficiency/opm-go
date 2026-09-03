---
description: Instructions for making unit tests for Go code
applyTo: "**/*.go"
---

Tests are ideally independent.
Do not use mocking.

Do not use loops. If there are multiple situations to check for a test, use the variable name to describe each.
Too many situations in a single test means it is doing too much. Break it up into multiple tests.

Tests must have 4 clearly labeled areas: Description, Arrange, Act, Assert.

A test failure in the "Arrange" means the setup isn't correct.
A test failure in "Act" means the code did not run correct.
A test failure in the "Assert" means things ran, but the results were not as expected.

All inputs in ACT must be declared in ARRANGE.
In ASSERT, if there are multiple things to verify, group them in logical sections.

Example:

```go
package calculator

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Add adds two integers together.
func Add(a, b int) int {
	return a + b
}

func TestAdd(t *testing.T) {
	// This test verifies that the Add function correctly calculates
	// the sum of two positive integers.

	// Arrange
	inputA := 5
	inputB := 7
	expected := 12

	// Act
	actual := Add(inputA, inputB)

	// Assert - No errors
	...

	// Assert - Correct structure
	...

	// Assert - Correct result
	assert.Equal(t, expected, actual)

}
```
