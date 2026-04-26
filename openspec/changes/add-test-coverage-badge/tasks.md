## 1. Coverage Calculation

- [ ] 1.1 Run `go test ./... -coverprofile=coverage.out` from the repository root and capture the resulting total coverage with `go tool cover -func=coverage.out`.
- [ ] 1.2 Decide whether the existing command documentation is enough or a small helper target/script is needed for repeatable coverage recalculation.
- [ ] 1.3 Ensure generated coverage profile artifacts such as `coverage.out` are not accidentally committed.

## 2. README Update

- [ ] 2.1 Add a visible test coverage percentage to the top overview or verification area of `README.md`.
- [ ] 2.2 Document the exact command or helper workflow used to recalculate the displayed coverage percentage.
- [ ] 2.3 Make the README wording explicit that the displayed value is static unless automatic publication is implemented later.

## 3. Verification

- [ ] 3.1 Run `go fmt ./...`.
- [ ] 3.2 Run `go test ./...`.
- [ ] 3.3 Re-run the documented coverage command and confirm the README percentage matches the calculated total.
- [ ] 3.4 Run OpenSpec validation/status checks for `add-test-coverage-badge` and confirm the change is apply-ready.
