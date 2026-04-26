## 1. Event Contract

- [ ] 1.1 Add task status changed event types with task ID, optional identifier/title fields, target state ID, and best available target state name.
- [ ] 1.2 Add a small in-process dispatcher/publisher interface for task manager status change events.
- [ ] 1.3 Add unit tests for dispatcher fan-out and handler error propagation.

## 2. Task Manager Integration

- [ ] 2.1 Update `Manager.MoveTask` to publish a status changed event only after Linear confirms a successful transition.
- [ ] 2.2 Add best-effort mapping from configured Linear state IDs to human-readable target state names.
- [ ] 2.3 Add unit tests proving successful transitions publish events, failed Linear transitions do not publish events, and handler failures are returned with task/state context.

## 3. Telegram Notification Handler

- [ ] 3.1 Add Telegram notification config fields and validation for API URL, bot token, and chat ID.
- [ ] 3.2 Implement Telegram `sendMessage` delivery with `net/http` and a testable HTTP sender.
- [ ] 3.3 Format status change messages with task identity and target state name, falling back to target state ID when needed.
- [ ] 3.4 Add unit tests for Telegram request payloads, fallback message content, and API/HTTP error handling.

## 4. Runtime Wiring

- [ ] 4.1 Load Telegram environment variables in centralized config.
- [ ] 4.2 Update `.env.example` with Telegram variable keys and no default values.
- [ ] 4.3 Register the Telegram handler in the CLI/runtime wiring when Telegram notification config is complete.
- [ ] 4.4 Ensure existing runs without Telegram config either skip handler registration or fail with an explicit configuration error only when Telegram notifications are enabled.

## 5. Documentation And Verification

- [ ] 5.1 Update `architecture.md` to describe status change events and Telegram notification boundaries.
- [ ] 5.2 Update or add user-facing docs for Telegram notification configuration if needed.
- [ ] 5.3 Run `go fmt ./...`.
- [ ] 5.4 Run `go test ./...`.
