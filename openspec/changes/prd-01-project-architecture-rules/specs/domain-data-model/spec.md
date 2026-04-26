## ADDED Requirements

### Requirement: Prisma schema defines geographic and taxonomy entities
The backend SHALL define Prisma models for `Country`, `City`, and `FieldOfStudy` with the fields and relationships required by DRO-12.

#### Scenario: Country model exists
- **WHEN** Prisma schema is inspected
- **THEN** `Country` includes `id`, `name`, `slug`, and `iso_code`

#### Scenario: City model belongs to country
- **WHEN** Prisma schema is inspected
- **THEN** `City` includes `id`, `name`, and `country_id` with a relation to `Country`

#### Scenario: Field of study model exists
- **WHEN** Prisma schema is inspected
- **THEN** `FieldOfStudy` includes `id`, `name`, and `slug`

### Requirement: Prisma schema defines university and program catalog
The backend SHALL define Prisma models for `University`, `AdmissionRequirements`, and `Program` with the fields and relationships required by DRO-12.

#### Scenario: University model exists
- **WHEN** Prisma schema is inspected
- **THEN** `University` includes `id`, `name`, `slug`, `description`, `website`, `logo`, `cover`, `tuition_min`, `tuition_max`, `currency`, `has_scholarships`, `country_id`, and `city_id`

#### Scenario: Admission requirements model exists
- **WHEN** Prisma schema is inspected
- **THEN** `AdmissionRequirements` includes `id`, `min_gpa`, `language_requirement`, `required_documents`, and `description`

#### Scenario: Program model links catalog entities
- **WHEN** Prisma schema is inspected
- **THEN** `Program` includes `id`, `name`, `level`, `language`, `duration`, `tuition_fee`, `currency`, `deadline`, `description`, `university_id`, `field_of_study_id`, and `admission_requirements_id`

#### Scenario: Program relations are navigable
- **WHEN** Prisma Client is generated
- **THEN** a program can be queried together with its university, field of study, and admission requirements

### Requirement: Prisma schema defines users, favorites, and leads
The backend SHALL define Prisma models for `User`, `Favorite`, and `Lead` with the fields and relationships required by DRO-12.

#### Scenario: User model exists
- **WHEN** Prisma schema is inspected
- **THEN** `User` includes `id`, `name`, `email`, `password_hash`, `phone`, and `created_at`

#### Scenario: Favorite model targets university or program
- **WHEN** Prisma schema is inspected
- **THEN** `Favorite` includes `id`, `user_id`, nullable `university_id`, and nullable `program_id`

#### Scenario: Lead model exists
- **WHEN** Prisma schema is inspected
- **THEN** `Lead` includes `id`, `name`, `email`, `phone`, `source_page`, nullable `university_id`, nullable `program_id`, `status`, and `created_at`

#### Scenario: Lead status is constrained
- **WHEN** Prisma schema is inspected
- **THEN** `Lead.status` is constrained to `new` or `processed`

### Requirement: Seed data populates local development database
The backend SHALL provide a deterministic Prisma seed that creates representative local data for all core catalog entities and lead/user examples.

#### Scenario: Seed creates countries cities and fields
- **WHEN** a developer runs the seed against an empty local database
- **THEN** it creates 5-10 countries, 10-20 cities, and 6-8 fields of study

#### Scenario: Seed creates universities and programs
- **WHEN** a developer runs the seed against an empty local database
- **THEN** it creates 20-30 universities and 50-100 programs connected to countries, cities, fields of study, and admission requirements

#### Scenario: Seed creates user and leads
- **WHEN** a developer runs the seed against an empty local database
- **THEN** it creates 1 test user and 3-5 leads

#### Scenario: Seeded data is visible in Prisma Studio
- **WHEN** a developer starts PostgreSQL, applies Prisma setup, runs the seed, and opens `npx prisma studio`
- **THEN** the seeded countries, cities, fields, universities, programs, user, favorites if present, and leads are visible

### Requirement: Prisma setup targets PostgreSQL 15
The backend Prisma datasource SHALL target PostgreSQL and use the database URL supplied through environment configuration.

#### Scenario: Prisma connects to local postgres
- **WHEN** PostgreSQL is running through `docker-compose up` and the backend environment is configured
- **THEN** Prisma migration, seed, and Studio commands connect to the PostgreSQL 15 service

#### Scenario: Prisma Client is generated
- **WHEN** a developer runs Prisma generate or install scripts that trigger generation
- **THEN** Prisma Client is generated from `backend/prisma/schema.prisma`
