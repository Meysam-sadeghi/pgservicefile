# Security Patches & Optimizations (`Meysam-sadeghi/pgservicefile`)

Base Upstream: `github.com/jackc/pgservicefile@5a60cdf`

| ID | Severity | CWE | File / Symbol | Summary |
|---|---|---|---|---|
| SEC-PGSERVICE-01 | MEDIUM | CWE-400 | `pgservicefile.go` (`ReadServicefile`, `ParseServicefile`) | Enforces `IsRegular()`, rejects `nil` readers, caps size to 4 MiB, line length to 64 KiB, sections to 10,000, and keys per service to 4,096. |
| SEC-PGSERVICE-02 | MEDIUM | CWE-158 / CWE-20 | `pgservicefile.go` (`ParseServicefile`) | Rejects NUL bytes on any line, empty section headers `[]`, and empty keys (`=value`). |
| SEC-PGSERVICE-03 | LOW | CWE-476 | `pgservicefile.go` (`Servicefile.GetService`) | Guards against `nil` receiver or uninitialized lookup map instead of panicking. |
| OPT-PGSERVICE-01 | OPT | — | `pgservicefile.go` (`ParseServicefile`) | Replaces `strings.SplitN(line, "=", 2)` with zero-allocation `strings.Cut(line, "=")`. |
