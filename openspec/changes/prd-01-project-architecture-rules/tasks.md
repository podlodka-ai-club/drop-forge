## 1. Project Scaffold

- [ ] 1.1 Create `/backend`, `/frontend`, and `/docs` directories with independent project boundaries.
- [ ] 1.2 Initialize `/backend` as a Next.js 14 TypeScript project configured for API routes only.
- [ ] 1.3 Initialize `/frontend` as a React 18 + Vite + TypeScript project.
- [ ] 1.4 Add Tailwind CSS configuration and base styles to `/frontend`.
- [ ] 1.5 Configure backend `npm run dev` to listen on port `3001`.
- [ ] 1.6 Configure frontend `npm run dev` to listen on port `3000`.

## 2. Runtime Configuration And Docker

- [ ] 2.1 Add root `docker-compose.yml` with PostgreSQL 15 service named `postgres`, host port `5432`, and persistent volume.
- [ ] 2.2 Update root `.env.example` with all required Docker, backend, frontend, and Prisma variable keys without values.
- [ ] 2.3 Add backend configuration loading for database and server runtime settings through environment variables.
- [ ] 2.4 Add frontend API client configuration that reads `VITE_API_URL`.
- [ ] 2.5 Verify source code does not hardcode secrets, environment URLs, database URLs, or runtime-specific credentials.

## 3. OpenAPI Contract And API Envelope

- [ ] 3.1 Create `docs/openapi.yaml` with API metadata, server definitions, common `{ data, error, meta }` response envelope schemas, and reusable error schemas.
- [ ] 3.2 Add an initial `/api/health` path to `docs/openapi.yaml`.
- [ ] 3.3 Add OpenAPI component schemas for the PRD-01 domain models that will be exposed by future endpoints.
- [ ] 3.4 Implement backend response helpers or conventions that produce `{ data, error, meta }` for success and error responses.
- [ ] 3.5 Implement `/backend/app/api/health/route.ts` according to `docs/openapi.yaml`.
- [ ] 3.6 Implement Swagger UI route at `/backend/app/api/docs` using `docs/openapi.yaml` as the displayed contract.

## 4. Prisma Data Model And Seed

- [ ] 4.1 Add Prisma dependencies and scripts to `/backend`.
- [ ] 4.2 Create `backend/prisma/schema.prisma` configured for PostgreSQL through environment variables.
- [ ] 4.3 Define `Country`, `City`, and `FieldOfStudy` models with required fields and relations.
- [ ] 4.4 Define `University`, `AdmissionRequirements`, and `Program` models with required fields and relations.
- [ ] 4.5 Define `User`, `Favorite`, and `Lead` models, including constrained lead status values.
- [ ] 4.6 Generate and commit the initial Prisma migration for the schema.
- [ ] 4.7 Add deterministic Prisma seed data for 5-10 countries, 10-20 cities, 6-8 fields, 20-30 universities, 50-100 programs, 1 test user, and 3-5 leads.
- [ ] 4.8 Verify `npx prisma studio` can display seeded local data after database setup.

## 5. Frontend Baseline

- [ ] 5.1 Add minimal React app structure that builds with Vite and Tailwind.
- [ ] 5.2 Add a small API availability check that calls the contract-defined health endpoint through `VITE_API_URL`.
- [ ] 5.3 Ensure frontend code depends only on the OpenAPI-defined endpoint shape and not backend implementation internals.

## 6. Documentation

- [ ] 6.1 Update root `README.md` with project structure and responsibility boundaries for backend, frontend, docs, Docker, and environment files.
- [ ] 6.2 Document local startup commands: `docker-compose up`, `cd backend && npm run dev`, and `cd frontend && npm run dev`.
- [ ] 6.3 Document Swagger UI availability at `http://localhost:3001/api/docs`.
- [ ] 6.4 Document Prisma setup, seed, and `npx prisma studio` usage.
- [ ] 6.5 Document the contract-first workflow for adding future API endpoints.

## 7. Verification

- [ ] 7.1 Run `go fmt ./...`.
- [ ] 7.2 Run `go test ./...`.
- [ ] 7.3 Run backend install and available backend checks such as lint, typecheck, build, Prisma generate, migration, and seed commands.
- [ ] 7.4 Run frontend install and available frontend checks such as lint, typecheck, and build.
- [ ] 7.5 Start PostgreSQL with `docker-compose up` and verify it accepts Prisma connections.
- [ ] 7.6 Start backend and verify `http://localhost:3001/api/docs` renders Swagger UI.
- [ ] 7.7 Start frontend and verify `http://localhost:3000` loads and can reach the backend health endpoint.
