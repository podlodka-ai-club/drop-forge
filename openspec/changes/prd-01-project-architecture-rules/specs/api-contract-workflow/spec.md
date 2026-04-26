## ADDED Requirements

### Requirement: OpenAPI is the source of truth for API endpoints
Every new API endpoint SHALL be described in `docs/openapi.yaml` before backend implementation and frontend consumption.

#### Scenario: New endpoint is proposed
- **WHEN** a developer starts work on a new API endpoint
- **THEN** the endpoint path, method, request shape, response shape, and error responses are added to `docs/openapi.yaml` before backend code is implemented

#### Scenario: Frontend consumes backend API
- **WHEN** frontend code calls an API endpoint
- **THEN** the call matches the path, method, request schema, response schema, and error schema defined in `docs/openapi.yaml`

### Requirement: API responses use a common envelope
All backend API responses SHALL use the JSON envelope `{ data, error, meta }`.

#### Scenario: Successful API response
- **WHEN** an API route completes successfully
- **THEN** the JSON response contains `data`, `error`, and `meta`, with `error` set to `null`

#### Scenario: Failed API response
- **WHEN** an API route returns a client or server error
- **THEN** the JSON response contains `data`, `error`, and `meta`, with `data` set to `null` and `error` containing machine-readable error information

#### Scenario: OpenAPI describes response envelope
- **WHEN** a response schema is added to `docs/openapi.yaml`
- **THEN** it describes the `{ data, error, meta }` envelope rather than a bare payload

### Requirement: Swagger UI is served by backend
The backend SHALL serve Swagger UI for `docs/openapi.yaml` at `http://localhost:3001/api/docs`.

#### Scenario: Developer opens API documentation
- **WHEN** the backend dev server is running and a developer opens `http://localhost:3001/api/docs`
- **THEN** Swagger UI renders the OpenAPI contract from `docs/openapi.yaml`

#### Scenario: Contract changes are visible
- **WHEN** `docs/openapi.yaml` is updated and the backend documentation route is refreshed
- **THEN** Swagger UI reflects the updated contract without requiring a separate documentation server

### Requirement: Backend and frontend depend only on the contract boundary
The frontend and backend SHALL interact through the OpenAPI contract and MUST NOT depend on each other's internal implementation details.

#### Scenario: Backend implementation changes internally
- **WHEN** backend code reorganizes Prisma calls, route helpers, or internal modules without changing `docs/openapi.yaml`
- **THEN** frontend code does not need to change

#### Scenario: Frontend implementation changes internally
- **WHEN** frontend code reorganizes components, hooks, or styling without changing API calls
- **THEN** backend code does not need to change

### Requirement: Minimal health endpoint verifies API availability
The initial contract SHALL include a minimal health endpoint that allows developers and the frontend to verify API availability without depending on product data.

#### Scenario: Developer checks backend availability
- **WHEN** a developer calls the health endpoint defined in `docs/openapi.yaml`
- **THEN** the backend responds with the common envelope and indicates that the API process is reachable
