# Development Notes

This file documents findings, limitations, and future considerations for the CS Demo Analyzer project.

## Failed Extraction Attempts

These are features or data points that we attempted to extract but were unable to due to technical limitations or API constraints.

### 🩸 Fall Damage extraction

**Status**: ❌ Failed / Not Feasible

**Reason**: 
The game events related to damage ([`player_hurt`]) capture damage from C4 explosions (`EqBomb`) but do not reliably categorize fall damage. Specifically, the damage information obtained from the demo parser often does not include `EqWorld` (World Entity) as weapon types in the contexts where fall damage would typically be expected, or the events themselves are missing for fall damage scenarios in certain demo recordings. This makes it difficult to distinguish fall damage from other world-based damage sources or to track it consistently.

For more details on the parser events, refer to the [demoinfocs-golang documentation](https://pkg.go.dev/github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events).

