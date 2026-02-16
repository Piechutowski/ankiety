# Ankiety — Project Conventions

Read STYLE.md first. It contains the personal coding philosophy and language-level conventions that apply here.

## What This Project Is

A Polish-language agricultural survey management system (BDGRoBMSP). Workers at accounting offices fill out surveys for farms. Managers oversee their office's workers. Admins see everything. Methodologists define survey structure.

See the project README for a domain glossary (IDGR, IDBR, IDPBR, etc.).

## Tech Stack

| Layer     | Technology                                             |
|-----------|--------------------------------------------------------|
| Backend   | Go 1.24, `net/http` stdlib (Go 1.22+ routing)         |
| Database  | SQLite via `mattn/go-sqlite3` + `jmoiron/sqlx`        |
| Sessions  | `alexedwards/scs/v2` (30-minute idle timeout)          |
| Forms     | `go-playground/form` (POST form decoding)              |
| Logging   | `log/slog` + `lmittmann/tint` (structured, colored)   |
| Frontend  | Vanilla TypeScript (strict mode), no framework         |
| CSS       | Tailwind CSS (utility-first)                           |
| Templates | Go `html/template` (server-side rendering)             |
| Assets    | Embedded via `//go:embed` — single binary deployment   |

No framework on either side. Backend uses Go stdlib + 3 libraries. Frontend is hand-written TypeScript.

## File Structure

```
main.go          All backend code: routes, handlers, middleware, DB manager,
                 template composition, helpers. This is the main file.
models.go        Data structs: DB models, template data types, table schema types.
static.go        Navigation tree (TabNode), system table definitions, methodology handlers.
main_test.go     Integration tests (httptest).

frontend/
  script.ts      All frontend logic: components, state, event handling, validation,
                 serialization, DOM utilities.
  script.js      Compiled output (do not edit).
  input.css      Tailwind source imports.
  output.css     Compiled Tailwind (do not edit).
  *.html         Go template fragments.

sql_master/      Queries for the master database (users, auth, access control).
sql_year/        Queries for year-specific databases (survey data, metadata).
sql_both/        Shared SQL (migrations, setup).

schema.dbml      Database schema documentation.

db/              SQLite database files (master.db, {year}.db). Not in git.
```

Do NOT create new Go source files unless there is a strong organizational reason. All backend code goes in `main.go`, `models.go`, or `static.go`. All frontend code goes in `script.ts`.

## Database Architecture

Two-tier SQLite design:

**Master database** (`master.db`): Users, roles, accounting offices (biura_rachunkowe), farms (gospodarstwa), year assignments. One instance, always open.

**Year databases** (`{year}.db`, e.g., `2024.db`, `2025.db`): Survey structure (tables, subtables, columns, codes, dictionaries) and survey data (b_bdgrobmsp stores JSON). One per active year, opened on demand.

`DBManager` manages both. It holds a single master connection and a `map[YearDB]*SqlCache` for year databases. Queries are pre-compiled into prepared statements at startup from `.sql` files, accessed by name:

```go
app.DBManager.MQueryx("user_data_get", login)                    // master
app.DBManager.YQueryx(yearDB, "b_kolumny_select_all_where_podtabela", subtable)  // year
```

## Routing

Uses Go 1.22+ `http.ServeMux` pattern matching with `{param}` path values.

Two middleware chains:
- `Logged` — requires authenticated session
- `AccessIdGR` — requires session + validates user has access to the specific farm (IDGR)

All routes are defined in `Application.Routes()`. Static assets (`/frontend/`) have separate caching headers.

## HTML Template Conventions

Templates are composed via `TmplCompose()` which combines multiple `html/template` fragments into a single template.

### Naming

Templates with the same layout share a prefix:
- `main_*` — main application layouts
- `table_*` — table rendering variants
- `input_*` — input type fragments

Template variables are `SCREAMING_SNAKE`: `TMPL_LOGIN`, `TMPL_GRID`, `TMPL_LIST_GR`.

### data-* Attributes

Used extensively for JS-DOM communication. Current conventions:

| Attribute                     | Purpose                                     |
|-------------------------------|---------------------------------------------|
| `data-table-type`             | Identifies table variant for JS init        |
| `data-table-statusy`          | Marks a status/list table                   |
| `data-endpoint`               | URL for saving/fetching                     |
| `data-cell`                   | Marks a table cell container                |
| `data-row-index`              | Row position for grouping cells             |
| `data-row-code`               | Domain code for a row                       |
| `data-enum-container`         | Wraps an enum dropdown component            |
| `data-enum-input`             | Visible search/display input for enum       |
| `data-enum-value`             | Hidden input holding the selected value     |
| `data-enum-dropdown`          | Dropdown option list container              |
| `data-enum-option`            | Single option in dropdown (`data-value`, `data-label`) |
| `data-multi-exclusive-*`      | Multi-select with exclusive option pattern  |
| `data-tooltip`                | Tooltip text content                        |
| `data-required`               | Field is required (`"true"`)                |
| `data-format`                 | Number or string format mask                |
| `data-min`, `data-max`        | Numeric bounds                              |
| `data-row-selector`           | Marks the add-row dropdown in dynamic tables|
| `data-row-adder`              | Input that triggers row addition            |
| `data-delete-row`             | Button to remove a dynamic row              |
| `data-initial`                | JSON string of existing data for dynamic tables |

No strict naming rule for new `data-*` attributes yet, but prefer `data-{component}-{role}` when the attribute is component-specific.

## Survey Table Types

The application renders 4 types of data tables, determined by `b_podtabele.schemat_tabeli`:

| Type                          | Description                                  |
|-------------------------------|----------------------------------------------|
| `HORIZONTAL_STATIC_UNIQUE`    | Fixed rows (one per code), horizontal layout  |
| `HORIZONTAL_DYNAMIC_UNIQUE`   | User adds rows from a list, each code once    |
| `HORIZONTAL_DYNAMIC_DUPLICABLE` | User adds rows, same code allowed multiple times |
| `VERTICAL_STATIC_UNIQUE`      | Each column becomes a row, single-value form  |

There is also `SYSTEM_DEFINITION` for admin/methodology system tables (mostly unimplemented).

## User Roles

Bitmask-based access control:

| Role             | Constant          | Access                                    |
|------------------|-------------------|-------------------------------------------|
| Admin            | `UserAdmin`       | All farms, all years, all features        |
| Methodologist    | `UserMethodolgist`| Survey structure editing (methodology)    |
| Manager (ZBR)    | `UserManager`     | Farms assigned to their accounting office |
| Worker (PBR)     | `UserNormal`      | Only their personally assigned farms      |

## Known Issues and TODOs

- **Passwords are stored in plaintext**. Salt field exists in schema but is unused. Must implement hashing (bcrypt/argon2) before production.
- **No CSRF protection** on POST endpoints.
- **No server-side input validation** — frontend validates, backend trusts authenticated users.
- **TLS config is defined but not enabled** — `ListenAndServe()` instead of `ListenAndServeTLS()`.
- **CSP header disabled** due to inline styles (`style` attribute set by JS for dropdown positioning).
- **Logging strategy is not finalized** — currently basic slog, needs structured approach.
- **Methodology module** — routes defined, handlers partially implemented.
- **System tables** — switch/case skeleton in `YearSystemTableCreate`, only `b_tabele` implemented.
- **Component reusability** — enum dropdown is used for both data input and row selection. These share UI mechanics but differ in behavior. Needs refactoring into primitive (dropdown) + application layer (what to do on select). See STYLE.md "Reusable UI Components" section.
