---
description: Rule for adding new features - requires standalone tests in tests/funData
---

# Feature Testing Rule

For every new feature added to the codebase, you **MUST** create a corresponding standalone test file.

## Requirements

1.  **Location**: The test file must be placed in `tests/funData/`.
2.  **Naming**: The file should be named descriptive to the feature, e.g., `feature_name_test.go`.
3.  **Content**:
    *   The test should be **standalone**, meaning it uses the parser to parse a demo file directly.
    *   It should **reuse** existing API constants (e.g., from `pkg/api/constants`) where possible to ensure consistency.
    *   It should implement the extraction/calculation logic **independently** from the main API implementation (where feasible) to serve as a cross-verification.
    *   It should print relevant data to stdout for manual inspection and include assertions to verify the logic.

## Example

See `tests/funData/wasted_utility_logic_test.go` for a reference implementation.

## Steps

1.  Plan the feature implementation.
2.  Implement the feature in `pkg/api`.
3.  Create the test in `tests/funData/`.
4.  Run the test using `go test -v ./tests/funData/your_test.go` (or package path).
5.  Verify the output matches expectations.
