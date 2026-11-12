---
name: bancolombia-clean-architecture
description: >
  Official architectural rules, Gradle scaffolding tasks, and best practices for Java/Spring Boot
  projects using the Bancolombia Clean Architecture Gradle Plugin (`co.com.bancolombia.cleanArchitecture`).
  Use this skill whenever initializing, planning, scaffolding, refactoring, or validating modules,
  domain models, gateway ports, use cases, driven adapters, entry points, helpers, or tests in a
  Bancolombia Clean Architecture repository.
license: Apache-2.0
metadata:
  plugin: co.com.bancolombia.cleanArchitecture
  scaffold-version: "5.x"
---

# Bancolombia Clean Architecture Gradle Plugin Skill

You are working in a multi-module Gradle project governed by the **Bancolombia Scaffold Clean Architecture Plugin** (`co.com.bancolombia.cleanArchitecture`).

Your primary directive is to **use the plugin's Gradle tasks to generate structural components** and **strictly enforce the inward Dependency Rule** across all layers.

---

## 1. Core Architecture & The Dependency Rule

Dependencies **MUST ALWAYS point inward**, from outer infrastructure/application layers toward the inner domain layers. Inner layers must never know anything about outer layers, frameworks, databases, or transport protocols.

```text
┌─────────────────────────────────────────────────────────────────────────┐
│  applications/app-service (Composition Root, MainApplication, Config)   │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │  infrastructure/ (entry-points, driven-adapters, helpers)         │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │  domain/usecase (Pure Business Orchestration)               │  │  │
│  │  │  ┌───────────────────────────────────────────────────────┐  │  │  │
│  │  │  │  domain/model (Entities, Value Objects, Gateway Ports)│  │  │  │
│  │  │  └───────────────────────────────────────────────────────┘  │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

### Scaffold v5+ Auto-Discovery (`SettingsConventionPlugin`)
- In Scaffold v5+, the `SettingsConventionPlugin` automatically discovers subprojects placed under `applications/`, `domain/`, and `infrastructure/` by scanning for their `build.gradle(.kts)` files and wires them into `app-service`.
- **NEVER** manually edit `settings.gradle` to add static `include` statements for new modules; always let the `./gradlew generate*` tasks create the module structure.

### Layer Responsibilities & Dependency Boundaries

| Gradle Module Path | Role & Contents | Allowed Dependencies | Strictly Forbidden |
| :--- | :--- | :--- | :--- |
| `domain/model` | Core domain entities, immutable Value Objects (Java 21 `record`), domain exceptions (`sealed interface` / runtime domain exceptions), and **Gateway interfaces** (`*Gateway` / `*Repository`). | Pure Java 21 + Project Reactor (`Mono`, `Flux` in reactive projects) **ONLY**. | Spring annotations (`@Service`, `@Component`, `@Repository`), JPA/R2DBC annotations (`@Entity`, `@Table`, `@Id`), Jackson annotations, Lombok (if avoidable in pure domain), or any external framework. |
| `domain/usecase` | Application business rules and orchestration (`*UseCase`). Receives domain models/commands and coordinates Gateway ports. | `domain/model` **ONLY**. | Spring stereotypes (`@Service`, `@Component`, `@Transactional`), HTTP/DB clients, DTOs, or direct references to any `infrastructure` module. |
| `infrastructure/driven-adapters/*` | Outbound implementations of `domain/model` Gateway interfaces (R2DBC, JPA, REST Consumer, Redis, Kafka/RabbitMQ, AWS S3/DynamoDB/SQS/Secrets). | `domain/model`, external SDKs/starters, `infrastructure/helpers/*`. | Direct dependencies on `applications/app-service` or `entry-points`. Never leak DB `@Entity` or external API DTOs into `domain`. |
| `infrastructure/entry-points/*` | Inbound adapters that trigger use cases (WebFlux Router/Handler, REST MVC Controllers, Kafka/SQS/MQ listeners, gRPC, GraphQL, MCP, A2A Agents). | `domain/model`, `domain/usecase`, web/messaging starters, `infrastructure/helpers/*`. | Direct dependencies on `driven-adapters` or `app-service`. Never put business rules in Handlers/Controllers. |
| `infrastructure/helpers/*` | Reusable technical utilities shared across infrastructure adapters (e.g., custom mappers, correlation ID extractors). | Utility libraries, `domain/model` (if needed). | Being imported by `domain/model` or `domain/usecase`. |
| `applications/app-service` | Bootable Spring Boot application (`MainApplication`), `UseCasesConfig` bean wiring, `SecurityFilterChain`/`SecurityWebFilterChain`, `ObjectMapper`, and `application.yaml`. | All `domain/*` and `infrastructure/*` modules (auto-wired by plugin conventions). | Business logic, domain entities, or direct repository queries. |

---

## 2. Mandatory Agent Workflow (Scaffold-First Protocol)

Whenever asked to build a new feature, integration, endpoint, or domain capability, follow this exact sequence:

### Step 1: Inspect Project Context
Before running tasks or writing code, read:
1. Root `build.gradle(.kts)` and `gradle.properties` to identify:
   - Base package (e.g., `co.com.bancolombia` or `com.company.service`).
   - Project paradigm: **Reactive** (`WebFlux` + `Mono`/`Flux`) vs. **Imperative** (`WebMVC`).
   - Java version (e.g., Java 21) and Spring Boot version.
2. Existing directories under `domain/` and `infrastructure/` to avoid duplicating adapters or models.

### Step 2: Generate Components Using Gradle Tasks
**NEVER** create Gradle module directories or `build.gradle` files manually. Always invoke `./gradlew` tasks from the service root directory:

1. **Create Domain Model & Gateway Port:**
   ```bash
   ./gradlew generateModel --name=<ModelName>
   ```
   *(Creates `domain/model/.../<modelname>/<ModelName>.java` and `gateways/<ModelName>Repository.java` or `<ModelName>Gateway.java`)*

2. **Create Use Case:**
   ```bash
   ./gradlew generateUseCase --name=<UseCaseName>
   ```
   *(Creates `domain/usecase/.../<usecasename>/<UseCaseName>UseCase.java` — note the plugin automatically appends `UseCase` if omitted)*

3. **Create Driven Adapter(s) (Outbound IO):**
   ```bash
   ./gradlew generateDrivenAdapter --type=<drivenAdapterType> [parameters]
   ```

4. **Create Entry Point(s) (Inbound Transport):**
   ```bash
   ./gradlew generateEntryPoint --type=<entryPointType> [parameters]
   ```

5. **Create Helper Module (Optional, Infrastructure-only):**
   ```bash
   ./gradlew generateHelper --name=<HelperName>
   ```

### Step 3: Implement Clean Code & Mappers
- Define clean domain contracts in `domain/model` first.
- Implement pure orchestration in `domain/usecase`.
- In `infrastructure`, implement **explicit bidirectional mapping** between Transport DTOs $\leftrightarrow$ Domain Models (in `entry-points`) and Domain Models $\leftrightarrow$ Persistence Entities / External Payloads (in `driven-adapters`).

### Step 4: Run Mandatory Verification Gates
Before considering any task complete, you **MUST** run:
```bash
./gradlew validateStructure
./gradlew test
```
And for full feature completion or PR readiness:
```bash
./gradlew build jacocoMergedReport
```

---

## 3. Complete Reference of Plugin Gradle Tasks

Use `./gradlew <task>` (or its short alias) with the exact flags supported by the plugin:

### 3.1 Project Initialization
| Task | Alias | Parameters | Example |
| :--- | :--- | :--- | :--- |
| `cleanArchitecture` | `ca` | `--package=<pkg>` `--type=<reactive\|imperative>` `--name=<ProjectName>` | `./gradlew ca --package=com.acme.orders --type=reactive --name=OrdersService` |

### 3.2 Domain Layer Tasks
| Task | Alias | Required Parameters | Generated Output |
| :--- | :--- | :--- | :--- |
| `generateModel` | `gm` | `--name=<PascalCaseName>` | Domain model class + gateway interface in `domain/model`. |
| `generateUseCase` | `guc` | `--name=<PascalCaseName>` | UseCase class in `domain/usecase` (auto-scanned as Spring Bean by `UseCasesConfig`). |

### 3.3 Driven Adapters (`generateDrivenAdapter` | `gda`)
Command syntax: `./gradlew generateDrivenAdapter --type=<type> [options]`

| `--type` Value | Purpose | Available Parameters & Defaults |
| :--- | :--- | :--- |
| `r2dbc` | Reactive PostgreSQL R2DBC client | — |
| `jpa` | Spring Data JPA repository | `--secret=<true\|false>` (default: `false`) |
| `mongodb` | MongoDB repository (reactive or imperative) | `--secret=<true\|false>` (default: `false`) |
| `dynamodb` | AWS DynamoDB adapter | — |
| `redis` | Redis template or repository adapter | `--mode=<template\|repository>` (default: `template`), `--secret=<true\|false>` |
| `restconsumer` | HTTP REST client (`WebClient` / `RestClient`) | `--url=<BaseUrl>`, `--from-swagger=<path/to/swagger.yaml>` |
| `rsocket` | RSocket requester adapter | — |
| `asynceventbus` | Async event publisher (Bancolombia Reactive Commons) | `--eda=<true\|false>` (default: `false`), `--tech=<rabbitmq\|kafka\|rabbitmq,kafka>` |
| `sqs` | AWS SQS message sender | — |
| `mq` | IBM MQ / JMS client sender | — |
| `s3` | AWS S3 storage adapter | — |
| `kms` | AWS KMS encryption adapter | — |
| `secrets` | Secrets Manager (AWS or HashiCorp Vault) | `--secrets-backend=<aws_secrets_manager\|vault>` (default: `aws_secrets_manager`) |
| `secretskafkastrimzi` | Kafka Strimzi credentials adapter | `--secret-name=<String>` |
| `cognitotokenprovider` | AWS Cognito OAuth2 token generator | — |
| `binstash` | BinStash multi-level cache | `--cache-mode=<LOCAL\|CENTRALIZED\|HYBRID>` (default: `LOCAL`) |
| `generic` | Empty custom driven adapter module | `--name=<KebabOrCamelName>` (Required) |

### 3.4 Entry Points (`generateEntryPoint` | `gep`)
Command syntax: `./gradlew generateEntryPoint --type=<type> [options]`

| `--type` Value | Purpose | Available Parameters & Defaults |
| :--- | :--- | :--- |
| `webflux` | Reactive REST API (Spring WebFlux) | `--router=<true\|false>` (default: `true`), `--authorization=<true\|false>`, `--versioning=<HEADER\|PATH\|NONE>`, `--from-swagger=<swagger.yaml>`, `--swagger=<true\|false>` |
| `restmvc` | Imperative REST API (Spring WebMVC) | `--server=<tomcat\|jetty>` (default: `tomcat`), `--authorization=<true\|false>`, `--versioning=<HEADER\|PATH\|NONE>`, `--from-swagger=<swagger.yaml>`, `--swagger=<true\|false>` |
| `asynceventhandler` | Async event/command/query listener | `--eda=<true\|false>`, `--tech=<rabbitmq\|kafka\|rabbitmq,kafka>` |
| `kafka` | Reactive/Imperative Kafka consumer | — |
| `kafkastrimzi` | Kafka Strimzi consumer | `--name=<String>`, `--topic-consumer=<TopicName>` |
| `sqs` | AWS SQS queue listener | — |
| `mq` | IBM MQ / JMS message listener | — |
| `graphql` | GraphQL API endpoint | `--pathgql=<String>` (default: `/graphql`) |
| `grpc` | gRPC Server (Spring gRPC + Protobuf) | — (Compile protos via `./gradlew :grpc:generateProto`) |
| `rsocket` | RSocket controller entry point | — |
| `mcp` | Model Context Protocol (MCP) Server | `--name=<String>`, `--enable-tools`, `--enable-resources`, `--enable-prompts`, `--enable-security`, `--enable-audit` (all default `true`) |
| `agent` | Spring AI A2A Reactive Agent | `--name=<String>`, `--agent-role=<collaborative\|supervisor\|hybrid>`, `--agent-enable-kafka=<true\|false>`, `--agent-enable-mcp-client=<true\|false>` |
| `generic` | Empty custom entry point module | `--name=<KebabOrCamelName>` (Required) |

### 3.5 Governance, Testing & Maintenance Tasks
| Task | Alias | Parameters | Purpose |
| :--- | :--- | :--- | :--- |
| `validateStructure` | `vs` | — | **Critical gate.** Verifies that no module violates Clean Architecture dependency rules. |
| `generateHelper` | `gh` | `--name=<HelperName>` | Creates a shared utility module under `infrastructure/helpers/`. |
| `deleteModule` | `dm` | `--module=<module-name>` | Cleanly deletes an existing Gradle submodule and its references. |
| `updateCleanArchitecture` | `u` | — | Migrates/updates the project to newer scaffold conventions and dependencies. |
| `generatePipeline` | `gp` | `--type=<github\|azure\|gitlab\|jenkins\|bitbucket\|circleci>` | Generates CI/CD pipeline definitions in `deployment/`. |
| `generateAcceptanceTest` | `gat` | `--name=<Name>` (default: `acceptanceTest`) | Scaffolds a Karate BDD acceptance test module in `deployment/`. |
| `generatePerformanceTest` | `gpt` | `--type=<jmeter>` | Scaffolds JMeter performance test plans in `deployment/`. |
| `jacocoMergedReport` | — | — | Runs tests across all modules and generates a unified JaCoCo coverage report. |

---

## 4. Coding Best Practices for Bancolombia Clean Architecture

### 4.1 Automatic UseCase Bean Wiring (`UseCasesConfig`)
- Do **NOT** annotate `*UseCase` classes with `@Service` or `@Component`.
- The scaffold provides `applications/app-service/src/main/java/.../config/UseCasesConfig.java` configured with `@ComponentScan` filtering classes matching `^.+UseCase$`.
- Keep `*UseCase` constructors clean (constructor injection via `final` fields) so Spring automatically instantiates and injects all required Gateway implementations at startup.

### 4.2 Reactive Best Practices (`--type=reactive`)
- **Zero Blocking Calls:** Never call `.block()`, `.blockFirst()`, `.blockLast()`, or `Thread.sleep()` inside `usecase`, `entry-points`, or `driven-adapters`.
- **Functional Routing:** When using `--type=webflux --router=true`, define routes declaratively in `RouterRest.java` (`RouterFunctions.route(...)`) and delegate HTTP request/response handling to `Handler.java`.
- **Error Handling:** Express business failures via domain exceptions (`Mono.error(new DomainBusinessException(...))`) and translate them centrally in `entry-points` (e.g., via `AbstractErrorWebExceptionHandler` returning RFC 7807 `ProblemDetail`).
- **Transactions:** Keep `@Transactional` or `TransactionalOperator` strictly inside `infrastructure/driven-adapters` (e.g., inside the R2DBC adapter method), never inside `domain/usecase`.
- **Reactive Testing:**
  - Test UseCases and Driven Adapters with `StepVerifier.create(...).expectNext(...).verifyComplete()`.
  - Test `RouterRest` + `Handler` with `@WebFluxTest` and `WebTestClient`.

### 4.3 Imperative Best Practices (`--type=imperative`)
- Use Java 21 `record` types for immutable commands, queries, and DTOs.
- Keep JPA `@Entity` classes inside `infrastructure/driven-adapters/jpa-repository` and convert them to `domain/model` classes inside the adapter before returning to `domain/usecase`.
- Test UseCases with JUnit 5 + Mockito (`@ExtendWith(MockitoExtension.class)`) and WebMVC controllers with `@WebMvcTest` + `MockMvc`.

### 4.4 Whitelisting Pure Dependencies in `domain/model` (If Ever Needed)
- By default, `validateStructure` fails if `domain/model` declares any external dependency.
- If an enterprise BOM or pure annotation artifact is strictly required, whitelist it in the root `build.gradle` rather than bypassing validation:
  ```groovy
  cleanPlugin {
      modelProps {
          whitelistedDependencies = "my-approved-bom"
      }
  }
  ```

---

## 5. Anti-Patterns & Red Lines (Strictly Prohibited)

1. **NEVER** create traditional layered packages (`controller/`, `service/`, `repository/`, `entity/`) inside `applications/app-service`.
2. **NEVER** import `org.springframework.*`, `jakarta.persistence.*`, `org.springframework.data.*`, or `com.fasterxml.jackson.*` inside `domain/model` or `domain/usecase`.
3. **NEVER** return a database entity (`*Entity`, `*Data`) or an infrastructure DTO (`*Request`, `*Response`) from a Gateway interface or a UseCase method.
4. **NEVER** bypass `./gradlew generateDrivenAdapter` or `./gradlew generateEntryPoint` by hand-crafting empty Gradle subprojects—the plugin configures dependencies, testing libraries, and module conventions automatically.
5. **NEVER** finish a task without running `./gradlew validateStructure` and ensuring exit code `0`.

---

## 6. Pre-Completion Verification Checklist

Before concluding your response after any code change:
- [ ] All new modules were scaffolded using `./gradlew generateModel`, `generateUseCase`, `generateDrivenAdapter`, `generateEntryPoint`, or `generateHelper`.
- [ ] `domain/model` contains only pure domain objects and Gateway interfaces.
- [ ] `domain/usecase` classes end with `*UseCase`, depend only on `domain/model`, and contain zero Spring annotations.
- [ ] All DTO $\leftrightarrow$ Domain and Entity $\leftrightarrow$ Domain conversions live inside `infrastructure`.
- [ ] `./gradlew validateStructure` passes with zero architectural violations.
- [ ] `./gradlew test` passes for all modified and newly generated modules.