## ADDED Requirements

### Requirement: Repository contains PRD-01 application structure
The repository SHALL contain the PRD-01 application structure with `/backend`, `/frontend`, `/docs`, `docker-compose.yml`, `.env.example`, and `README.md` at the repository root.

#### Scenario: Developer inspects repository layout
- **WHEN** a developer lists the repository root after the change is implemented
- **THEN** the root contains `backend/`, `frontend/`, `docs/`, `docker-compose.yml`, `.env.example`, and `README.md`

#### Scenario: OpenAPI contract is discoverable
- **WHEN** a developer opens `docs/`
- **THEN** `docs/openapi.yaml` exists as the API contract entrypoint

### Requirement: Backend project runs independently
The backend SHALL be an independent Next.js 14 project that exposes API routes only from `/backend/app/api/` and starts on port `3001` with `npm run dev` from `/backend`.

#### Scenario: Backend development server starts
- **WHEN** a developer runs `npm run dev` inside `/backend`
- **THEN** the backend development server listens on `http://localhost:3001`

#### Scenario: Backend uses API routes directory
- **WHEN** backend API code is added
- **THEN** route handlers live under `/backend/app/api/`

#### Scenario: Backend has no page UI requirement
- **WHEN** the backend project is inspected for product UI pages
- **THEN** the backend contains no required user-facing Next.js pages for the PRD-01 frontend experience

### Requirement: Frontend project runs independently
The frontend SHALL be an independent React 18 + Vite + Tailwind CSS project and start on port `3000` with `npm run dev` from `/frontend`.

#### Scenario: Frontend development server starts
- **WHEN** a developer runs `npm run dev` inside `/frontend`
- **THEN** the Vite development server listens on `http://localhost:3000`

#### Scenario: Tailwind is available to frontend
- **WHEN** frontend source files are built or served
- **THEN** Tailwind CSS styles are configured and available to React components

### Requirement: README documents local readiness workflow
The root `README.md` SHALL document the local readiness workflow for Docker, backend, frontend, Swagger UI, and Prisma Studio.

#### Scenario: Developer follows startup instructions
- **WHEN** a developer follows the README startup steps
- **THEN** the instructions cover `docker-compose up`, `cd backend && npm run dev`, `cd frontend && npm run dev`, Swagger UI at `http://localhost:3001/api/docs`, and `npx prisma studio`

#### Scenario: Developer checks project responsibilities
- **WHEN** a developer reads the README
- **THEN** it explains that the backend, frontend, database, OpenAPI contract, and environment template have separate responsibilities

### Requirement: PostgreSQL runs through docker compose
The repository SHALL provide `docker-compose.yml` with a `postgres` service using PostgreSQL 15, host port `5432`, and a persistent Docker volume for database data.

#### Scenario: Developer starts database
- **WHEN** a developer runs `docker-compose up`
- **THEN** a PostgreSQL 15 service named `postgres` starts and exposes port `5432`

#### Scenario: Database data persists
- **WHEN** the PostgreSQL container is recreated without deleting volumes
- **THEN** database data persists in the configured Docker volume
