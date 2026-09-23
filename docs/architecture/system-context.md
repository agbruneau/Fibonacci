# System Context — C4 Level 1

FibCalc seen from the outside: the user, and the three external systems the binary touches (optional GMP, OS, file system).

```mermaid
C4Context
    title System Context Diagram — FibCalc (C4 Level 1)

    Person(user, "User", "Developer or researcher computing Fibonacci numbers")

    System(fibcalc, "FibCalc", "High-performance Fibonacci calculator with CLI and TUI modes")

    System_Ext(gmp, "GMP Library", "GNU Multiple Precision Arithmetic (optional, build tag)")
    System_Ext(os, "Operating System", "CPU/memory metrics, signal handling")
    System_Ext(fs, "File System", "Calibration profile persistence (~/.fibcalc_calibration.json)")

    Rel(user, fibcalc, "Runs CLI commands / interacts with TUI")
    Rel(fibcalc, gmp, "Uses for GMP-accelerated fast doubling (optional)")
    Rel(fibcalc, os, "Reads CPU/memory stats, handles signals")
    Rel(fibcalc, fs, "Reads/writes calibration profiles")
```

---
[← Back to the architecture hub](./README.md)
Narrative legend of this figure: [§1 Project Overview](../ARCH.md#1-project-overview).
