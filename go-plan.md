# Go Enterprise Sprint — June 1 to 25

One plan, day by day. Each task says **what** to do, **How:** to actually do it, and what "done" looks like.

**Budget:** ~3 hrs/day, weekdays (weekends = catch-up/rest). 19 working days, ending Thu Jun 25.
**Locked scope:** (1) one HTTP service done properly — enterprise structure + OAuth/OIDC + Postgres; (2) a second **gRPC** service that Service A calls; (3) a real **CI/CD** pipeline.
**Stretch (extra time bought these):** distributed tracing (Day 13) and an optional Kubernetes deploy (Day 18).

**How to use this plan**
- You learn the **language just-in-time** by building, not by studying. The only abstract study is Day 2 (the three idioms that trip up Java devs).
- Dependency wiring is **manual** — no DI framework this sprint.
- One IdP (**Keycloak**) and one DB (**Postgres**), both run locally in containers.
- If you slip, cut in this order: Kubernetes → tracing → extra tests. Protect structure, OAuth, gRPC, and the pipeline.

### The toolchain — you learn these by using them, not by memorizing them
You asked how to "learn `go run`/`go fmt`" etc. You don't sit down and study them; each one shows up the day you first need it:

| Command | When you meet it | What it does |
|---|---|---|
| `go mod init <module>` | Day 1, creating the repo | Creates `go.mod` (your `pom.xml`) |
| `go mod tidy` | Every time you add/remove an `import` | Syncs dependencies to actual imports — run it, commit the result |
| `go run ./cmd/api` | Day 1+ | Fastest way to run during dev |
| `go build -o bin/api ./cmd/api` | Day 14 (Docker) | Produces the static binary you ship |
| `go test -race -cover ./...` | Day 9 onward | Runs tests; `-race` finds data races |
| `go vet ./...` | Wired into Makefile/CI Day 1 | Flags suspicious code; just runs automatically |
| `gofmt` / format-on-save | Day 1, set once | Canonical formatting — never think about it again |
| `go doc <pkg>` / pkg.go.dev | Whenever you need an API | Your reference habit instead of guessing |
| `go generate ./...` | Day 5 (sqlc), Day 10 (buf) | Runs code generators |

---

## The enterprise stack (in scope — ignore alternatives for now)
router `chi` · config `caarlos0/env` · db `pgx` + `sqlc` · migrations `golang-migrate` · auth `golang.org/x/oauth2` + `coreos/go-oidc` · logging `log/slog` · tracing OpenTelemetry · metrics `prometheus/client_golang` · rpc gRPC + `buf` · tests `testing` + `testify` + `testcontainers-go` · lint `golangci-lint` · vuln `govulncheck` · CI GitHub Actions · containers multi-stage Docker → distroless.

## Java → Go cheat sheet (keep this open)
| Java / Spring | Go |
|---|---|
| Class + methods | `struct` + methods with receivers |
| Inheritance | Composition + embedding (no `extends`) |
| `interface implements X` | Implicit interfaces — satisfied just by having the methods |
| Exceptions / `try-catch` | `error` return values + `defer` |
| `@Autowired` / container | Manual constructor wiring in `main()` |
| `@Service`/`@Repository` | Plain structs + packages |
| `null` / `Optional<T>` | zero values, `nil`, comma-ok `(T, bool)` |
| Threads / `CompletableFuture` | goroutines + channels |
| `ThreadLocal` / request scope | `context.Context` as first arg |
| Lombok annotations | struct tags (`json:"..."`) |
| Spring Security | Hand-built middleware + token verification |

---

# Week 1 (Jun 1–5) — Foundations and your first working slice

## Day 1 — Mon Jun 1: Repo, toolchain-by-doing, first running server
- [x] **Set up Go + IDE.** How: install the latest Go; in GoLand or VS Code (+ Go extension) enable `gopls`, format-on-save, and inline "run test" buttons. The red squiggles and quick-fixes are how you'll actually pick up syntax.
- [x] **Create the module.** How: `go mod init github.com/you/identity-api`. This writes `go.mod`. Any import you add later, run `go mod tidy` and commit the change.
- [x] **Write and run a tiny HTTP server.** How: a `main.go` that starts `http.ListenAndServe(":8080", ...)` returning `"ok"`; run with `go run .`; hit it with `curl localhost:8080`. You now know `go run` and the shape of `net/http`.
- [x] **Add a Makefile + linter.** How: targets `run`, `build`, `test`, `lint`, `migrate`; install `golangci-lint`, add `.golangci.yml`, run `make lint`. From now on vet/fmt/lint happen for you (on save + in CI) so you stop thinking about them.
- **Done:** `make run` serves a request; `make lint` is green.

## Day 2 — Tue Jun 2: The three idioms that make Go feel un-Java
Use the **Tour of Go** as a lookup, not a course. Write small throwaway snippets to drill these.
- [x] **Errors as values.** How: write a func returning `(User, error)`; handle with `if err != nil`; wrap with `fmt.Errorf("loading user %d: %w", id, err)`; inspect with `errors.Is`. Define `var ErrNotFound = errors.New("not found")`. Pick your convention: wrap at boundaries, return sentinels for known cases.
- [x] **Implicit interfaces + structs.** How: define a struct, add methods with a pointer receiver (`func (s *Store) Get(...)`), define a 1-method interface, and notice you never write "implements". Build a `UserRepo` interface with one in-memory implementation — this is the seam you'll swap for Postgres later.
- [x] **`context.Context`.** How: understand it carries cancellation + deadlines; you'll pass `ctx context.Context` as the first arg to anything doing I/O. Write a function that returns early when `ctx.Done()` fires.
- [x] **Pointers, zero values, slices/maps (quick drills).** How: `*T` receiver for mutation/large structs, `T` otherwise; a `nil` map panics on write but a `nil` slice appends fine; every type has a usable zero value. 30 minutes, then move on. (Goroutines/channels: skip — you'll meet them via the HTTP server and gRPC.)
- **Done:** you can read idiomatic Go and explain why there's no `try/catch` or `implements`.

## Day 3 — Wed Jun 3: Project structure + the composition root (the Spring-replacement day)
- [x] **Lay out the directories.** How: `cmd/api/main.go` (entrypoint); `internal/` for private packages organized **by domain** — `internal/user`, `internal/auth`, `internal/store` (DB), `internal/httpapi` (handlers). `internal/` is compiler-enforced private. Domain packages beat `controllers/services/repos` because the compiler forbids circular imports, which keeps boundaries honest.
- [x] **Define the seams (ports & adapters).** How: domain code depends on interfaces (`UserRepo`, `TokenVerifier`), not on `pgx` or `oauth2` directly. Those are adapters at the edges.
- [x] **Write the composition root.** How: in `main()`, build in order — config → db pool → repositories → services → handlers → server — passing each into the next constructor: `svc := user.NewService(repo, logger)`. *This is your DI.* If something isn't wired, it won't compile.
- [x] **Typed config from env.** How: a `Config` struct with `env:"DB_URL"` tags via `caarlos0/env`; load and validate at startup; fail fast with a clear message if a required var is missing.
- **Done:** the app boots from `main()` with everything wired by hand; missing config aborts startup loudly.

## Day 4 — Thu Jun 4: HTTP layer — chi, middleware, graceful shutdownf
- [x] **Router.** How: add `chi`; register routes (`r.Get("/healthz", h.Health)`); use route groups for versioning and the auth-protected section.
- [x] **JSON + error envelope.** How: write two helpers — `decode(r, &dst)` and `respond(w, status, v)` — and a consistent error shape like `{"error":{"code":"...","message":"..."}}`. Handlers stay thin.
- [x] **Middleware chain.** How: middleware is `func(http.Handler) http.Handler`. Add (in order) request ID → `slog` request logging → panic recovery → CORS. Order matters: recovery must wrap the handlers it protects.
- [x] **Graceful shutdown.** How: run `srv.ListenAndServe()` in a goroutine; catch SIGINT/SIGTERM; call `srv.Shutdown(ctx)` with a timeout so in-flight requests drain instead of being killed.
- **Done:** `/healthz` + one resource endpoint served through the full middleware chain; Ctrl-C shuts down cleanly.

## Day 5 — Fri Jun 5: Postgres — pgx + sqlc + migrations (first DB-backed slice)
- [x] **Run Postgres locally.** How: `docker run` Postgres (you'll fold it into compose on Day 15); create your database.
- [x] **Migrations.** How: `golang-migrate`; write `0001_init.up.sql` / `.down.sql` (a `users` table); run via `make migrate`. This is your Flyway equivalent — versioned schema in git.
- [x] **Type-safe queries with sqlc.** How: put SQL in `query.sql`, configure `sqlc.yaml`, run `sqlc generate` (that's `go generate`). It emits Go functions with typed params and results — no ORM, no runtime surprises.
- [x] **Implement the repository.** How: make your `UserRepo` interface concrete using the generated code via `pgx`; pass `ctx` to every query; do one write inside a transaction.
- [x] **Wire the vertical slice.** How: repo → service → handler so a real HTTP request reads/writes Postgres.
- **Done:** a `POST` then `GET` round-trips through the DB. First full slice complete.

---

# Week 2 (Jun 8–12) — OAuth, hardening, tests, start gRPC

## Day 6 — Mon Jun 8: OAuth/OIDC — the login flow
- [x] **Run Keycloak.** How: official Keycloak Docker image; in the admin console create a realm, a client (public + PKCE, or confidential), and a test user. Note the discovery URL `…/.well-known/openid-configuration`.
- [x] **Build the OAuth2 client.** How: `golang.org/x/oauth2` `Config` (client ID/secret, endpoints, redirect URL, scopes including `openid`); generate a random `state` + PKCE verifier; redirect the browser to the auth URL.
- [x] **Handle the callback.** How: verify `state` matches; exchange `code` for tokens with `config.Exchange(ctx, code)`.
- [x] **Verify the ID token (OIDC).** How: build a `coreos/go-oidc` provider from the issuer URL; verify the ID token (signature via JWKS, issuer, audience, expiry); read claims (`sub`, `email`).
- **Done:** you log in through Keycloak and land back in your app with a verified identity.

## Day 7 — Tue Jun 9: Protecting the API
- [x] **Establish your own session/token.** How: after login, either set a secure session cookie or mint a short-lived app JWT. Pick one and implement it.
- [x] **Auth middleware.** How: read the bearer token; verify against the provider's JWKS (cache the keys); check `exp`/`iss`/`aud`; put the principal (`sub`, roles) into the request `context`; otherwise return 401.
- [x] **Authorization.** How: a small middleware/helper that checks roles or claims and returns 403 when missing.
- [x] **Security hygiene.** How: cookies `HttpOnly`+`Secure`+`SameSite`; all secrets from env; run `govulncheck ./...` and fix anything it flags.
- **Done:** a protected endpoint returns data with a valid token and 401/403 without.

## Day 8 — Wed Jun 10: Buffer + consolidation
- [ ] **Finish OAuth + refactor.** How: OAuth almost always overruns Days 6–7 — this is your slack. Tidy the auth package, extract the token-verifier behind an interface (so it's mockable on Day 9), and make sure the middleware chain reads cleanly.
- [ ] **Catch up on any earlier slip** (DB transactions, error envelope, config validation).
- **Done:** Service A is feature-complete for auth + persistence and you're not behind.

## Day 9 — Thu Jun 11: Tests for Service A
- [ ] **Table-driven tests.** How: the dominant Go pattern — a slice of `{name, input, want}` cases looped with `t.Run`. Write them for the auth middleware (valid / expired / wrong-audience tokens) and a repo function.
- [ ] **Hand-written fakes.** How: because interfaces are implicit, write a tiny fake `UserRepo` and a fake `TokenVerifier` for service tests — no mock framework needed (reach for `gomock`/`mockery` only if it gets repetitive).
- [ ] **HTTP tests.** How: `net/http/httptest` `ResponseRecorder` to exercise a handler end-to-end without opening a real port.
- [ ] **Integration test.** How: `testcontainers-go` starts an ephemeral Postgres; run your migrations against it; test the repository for real. Run everything with `go test -race ./...`.
- **Done:** `make test` is green with one real-DB integration test; you now know `go test`/`-race`/`-cover` by using them.

## Day 10 — Fri Jun 12: gRPC Service B (server) with buf
- [ ] **Define the contract.** How: write a small `.proto` (e.g. a `Profile` service with one method + messages); manage it with `buf` (`buf.yaml`, `buf.gen.yaml`), then `buf lint` and `buf generate` to emit Go stubs. `buf` is the enterprise-standard proto workflow.
- [ ] **Implement the server.** How: a new `cmd/profile/main.go`; implement the generated server interface backed by a trivial store; serve gRPC on its own port.
- [ ] **Add an interceptor.** How: a unary logging + panic-recovery interceptor — gRPC's equivalent of HTTP middleware.
- **Done:** Service B serves the gRPC method (verify with `grpcurl` or a quick client).

---

# Week 3 (Jun 15–19) — Inter-service, observability, containers

## Day 11 — Mon Jun 15: Wire A → B over gRPC
- [ ] **gRPC client in Service A.** How: dial Service B at startup (inject the client via the composition root); call the method with a `context` deadline; map gRPC status codes to your HTTP error envelope.
- [ ] **Propagate context across the boundary.** How: pass the request ID and identity from A to B via gRPC **metadata** so logs from both services correlate.
- [ ] **End-to-end slice.** How: an authenticated request to A fetches profile data from B and returns combined JSON.
- **Done:** login → A → gRPC → B → combined response works locally.

## Day 12 — Tue Jun 16: Structured logging + metrics
- [ ] **Structured logging with `slog`.** How: configure a JSON handler at startup in both services; build a per-request logger carrying the request/trace ID; log key events (not everything). (Many existing enterprise codebases use `zap`/`zerolog` — you'll recognize them; `slog` is the modern default.)
- [ ] **Metrics.** How: `prometheus/client_golang`; expose `/metrics`; add a request counter and a latency histogram per service.
- **Done:** both services emit correlated JSON logs and expose Prometheus metrics.

## Day 13 — Wed Jun 17: Distributed tracing (the extra-time payoff)
- [ ] **OpenTelemetry setup.** How: add the OTel SDK to both services; configure a tracer provider and an OTLP exporter (point it at a local Jaeger/Tempo container, or stdout to start).
- [ ] **Instrument the seams.** How: wrap the HTTP handler and the gRPC client/server with OTel middleware/interceptors so spans propagate automatically; add a span around the DB call.
- [ ] **Verify one trace spans A → B.** How: make an authenticated request and confirm a single trace shows the HTTP entry, the gRPC call, and the DB query.
- **Done:** one request produces one distributed trace across both services. (If you're behind, this is the first thing to drop.)

## Day 14 — Thu Jun 18: Container images
- [ ] **Multi-stage Dockerfile per service.** How: stage 1 uses a `golang` image to compile a static binary (`CGO_ENABLED=0 go build -o /app ./cmd/api`); stage 2 is `distroless` (or `scratch`) and copies only the binary, running as non-root. Tiny image, minimal attack surface — this is where `go build` flags matter.
- [ ] **Build and run each image locally.** How: `docker build`, `docker run`, hit the endpoints to confirm parity with `go run`.
- **Done:** both services run from small, non-root images.

## Day 15 — Fri Jun 19: docker-compose — the whole system locally
- [ ] **Compose file.** How: services for `api`, `profile`, `postgres`, `keycloak` (+ Jaeger if you did Day 13); shared network, env vars, `depends_on` with healthchecks; one `make up`.
- [ ] **Verify the full flow in containers.** How: login via Keycloak → call A → A calls B → combined response, with logs/metrics/trace visible.
- **Done:** `make up` brings up the entire system; this is your sprint "deploy target".

---

# Week 4 (Jun 22–25) — CI/CD, optional deploy, capstone

## Day 16 — Mon Jun 22: CI pipeline (GitHub Actions)
- [ ] **Build the workflow.** How: on push/PR run — checkout → `actions/setup-go` (with `cache: true`) → `go vet ./...` → `golangci-lint-action` → `govulncheck ./...` → `go test -race -coverprofile=cover.out ./...` → `go build ./...`.
- [ ] **Add caching + branch protection.** How: module/build cache keeps CI fast; require these checks to pass before merge, no merge on red.
- **Starter skeleton:**
```yaml
name: ci
on:
  push: { branches: [main] }
  pull_request:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: 'stable', cache: true }
      - run: go vet ./...
      - uses: golangci/golangci-lint-action@v6
      - run: go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...
      - run: go test -race -coverprofile=cover.out ./...
      - run: go build ./...
```
- **Done:** every push runs the full gate; red blocks merge.

## Day 17 — Tue Jun 23: CD — images, supply chain, deploy step
- [ ] **Build and push images.** How: a job that builds both Docker images, tags them with the git SHA (and `latest` on main), and pushes to GHCR.
- [ ] **Supply-chain basics.** How: Trivy image scan in CI; `syft` SBOM; optionally `cosign` to sign images.
- [ ] **Deploy stage to your infra.** How: minimum is "image is in the registry, ready to pull". If you have a target (a VM via SSH, or a managed container service), add a deploy-on-tag job. Real cluster deploy is the optional Day 18.
- **Done:** a merge to main produces a scanned, tagged image; releases are reproducible.

## Day 18 — Wed Jun 24: OPTIONAL — minimal Kubernetes deploy (or buffer)
*You didn't lock k8s, so this is yours to take or swap for hardening/buffer.*
- [ ] **Local cluster.** How: spin up `kind` (or minikube); learn just enough `kubectl` (pods, deployments, services, configmaps, secrets).
- [ ] **Deploy both services.** How: a small Helm chart (or plain manifests) per service — deployment + service + configmap/secret, liveness/readiness probes hitting your health endpoints, resource requests/limits; wire A→B via the in-cluster Service DNS name.
- [ ] **If skipping k8s:** spend today hardening — tighten timeouts, validate all input, review error responses for leaks, raise test coverage on the auth path.
- **Done:** either both services run in `kind` reachable via a Service, or Service A is noticeably more robust.

## Day 19 — Thu Jun 25: Capstone wrap + the lead deliverable
- [ ] **All gates green.** How: `golangci-lint`, `go test -race ./...`, `govulncheck` all passing in CI.
- [ ] **READMEs.** How: one per service — architecture sketch, how to run (`make up`), how to test.
- [ ] **Write the team blueprint (the highest-leverage artifact).** How: a one-pager — "Go conventions & service blueprint": directory layout, the in-scope stack, error-handling convention, testing approach, CI gates, how OAuth is wired. This is what carries straight into your real project and standardizes your team.
- **Done:** two working services, a green pipeline, and a document your team can build the next service from.

---

## Reality check
At ~3 hrs/day this is full but achievable, with genuine slack: Day 8 is a real buffer, and the cut order (Kubernetes → tracing → extra tests) protects your three locked must-haves. OAuth (Days 6–8) and the A→B gRPC wiring (Days 10–11) are where slippage concentrates — watch those.

## Cut for after the sprint (your "go deeper" list)
Deep concurrency patterns (worker pools, fan-in/out), DI frameworks (`google/wire`, `uber/fx`), advanced generics, contract-first REST with `oapi-codegen`, full GitOps (Argo CD / Flux), and production Kubernetes (HPA, ingress, secrets management via Vault).

## Resources
*A Tour of Go*, *Effective Go*, *Go by Example*, and `pkg.go.dev` (read package docs directly). Books: Alex Edwards' *Let's Go* + *Let's Go Further* (structuring a real backend), Teiva Harsanyi's *100 Go Mistakes and How to Avoid Them*. Verify current library and IdP/registry versions when you reach the auth, tracing, and CI phases — those ecosystems move.
