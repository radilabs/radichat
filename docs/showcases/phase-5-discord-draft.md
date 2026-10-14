# Draft Discord showcase post — Owner review only

RadiChat's Step 5 Build Week adversarial investigation is complete and ready for review.

We used the accepted `radichat-mock` and a build-tagged Go harness to exercise streaming fragmentation, malformed and incomplete SSE, HTTP failures, authentication and redaction, redirects, and context-accounting boundaries. The corrected documented matrix recorded **63 PASS, 17 optional cases not run with reasons, 3 exploratory observations, and zero confirmed divergences**. No product fix was needed or made.

Early calibration surfaced two apparent divergences caused by test setup: Go's mock response added a sniffed `text/plain` content type, and two exact-fit budget values were one byte too small. Both candidates were withdrawn. A separate early test skip also hid a sibling subcase and was corrected. The final matrix passed. This shows why surprising test results need independent review before becoming defect claims.

The work kept role boundaries: the Architect defined the documented test matrix, Step 5 Preview ran the investigation, Luna independently checked additional cases, Dr Watson reviewed the harness and requested stronger assertions, and the existing Watcher independently verified the accepted candidate. The public test summary includes commands to reproduce the tracked matrix and its results:
https://github.com/radilabs/radichat/blob/main/docs/testing-phase-5.md

Seventeen optional cases were left out to keep the investigation bounded. Three undocumented probes remain observations, not defect claims. No Phase 6 work or product fixes are authorized. This post is a draft; publication awaits Owner approval.
