# Personal Style Guide

## Philosophy

I follow the Handmade Philosophy. Casey Muratori, Jonathan Blow, GingerBill, Eskil Steenberg — these are people who code well and differently from mainstream. I am a member of the Handmade community.

Build from the simplest primitives the language provides. If the standard library gives you what you need, use it. A library is justified only when the language truly cannot do the job (e.g., SQLite bindings, session management). A framework is almost never justified.

File Pilot is built in 5 files where one has 70K LOC. 2K lines in a file is nothing. Do NOT split files for the sake of splitting. Split only when there is a genuine organizational boundary.

## Anti-Patterns — Do NOT

- Do not apply SOLID principles
- Do not use OOP. No inheritance hierarchies, no polymorphism for its own sake. Use structs and procedures
- Do not reach for microservices. A monolith that compiles to one binary is the goal
- Do not add frameworks. Build on language primitives and standard library
- Do not containerize by default. Containers wrap the kernel — if the kernel already provides the feature, use it directly
- Do not use server-based services (PostgreSQL, Redis) when an in-process library (SQLite) will do. Fewer moving parts, fewer updates, fewer failure modes
- Do not add abstractions "for the future." Three similar lines are better than a premature helper
- Do not add error handling, fallbacks, or validation for scenarios that cannot happen
- Do not write classes. Write data (structs/types) and procedures (functions) that operate on that data
- Do not create interfaces with a single implementation
- Do not add comments that restate the code. Comments explain *why*, not *what*

## General Principles

- Prefer explicit over implicit
- Prefer simple over clever
- Prefer data-oriented design — think about how data flows, not about object hierarchies
- Prefer composition of simple functions over abstraction layers
- Strict typing always. Dynamic typing is for prototypes, not programs
- Minimize dependencies. Every dependency is a liability
- Read and understand what you depend on

## Go Conventions

Follow standard Go conventions with one addition:

- Exported types, functions, methods: `CamelCase` (standard Go)
- Unexported types, functions, methods: `camelCase` (standard Go)
- Package-level constants and global variables: `SCREAMING_SNAKE_CASE`
- Examples: `FS_FRONTEND`, `TMPL_LOGIN`, `SAVE_COOLDOWN_MS`

Everything else follows idiomatic Go — `error` returns, short variable names in small scopes, receiver methods on the `Application` struct.

## TypeScript Conventions

Follow Odin language naming conventions. In general, Ada_Case for types and snake_case for values.

| Element              | Case                    | Example                    |
|----------------------|-------------------------|----------------------------|
| Import name          | snake_case (prefer single word) | —                   |
| Types                | Ada_Case                | `User_Menu`, `Table`, `Number_Format` |
| Enum values          | Ada_Case                | `Horizontal_Static_Unique` |
| Procedures/functions | snake_case              | `table_init`, `user_menu_click` |
| Local variables      | snake_case              | `is_visible`, `row_index`  |
| Constants            | SCREAMING_SNAKE_CASE    | `SAVE_COOLDOWN_MS`         |

Additional rules:
- Strict TypeScript config: `strict: true`, `noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`
- No classes. Use type definitions (structs) and standalone functions
- Functions that belong to a type use the type name as prefix: `table_init`, `table_save`, `table_validate_all`
- No `this`. State is always passed as the first argument

## Frontend Architecture

The browser is a retained-mode GUI. I work with it, not against it. No immediate-mode wrappers like React.

### Event-Driven Model

Every page load follows two phases:

1. **Init phase**: `DOMContentLoaded` fires. Scripts query the DOM, build state structs, wire event listeners. After init, the page is interactive.
2. **Interaction phase**: User actions fire events. Handlers receive events, read/modify state, update the DOM.

### Component Pattern

Every UI component follows this structure:

```typescript
// 1. State type — plain data, no methods
type Logout = {
    button: HTMLElement;
    progress: HTMLElement;
    form: HTMLFormElement;
    timer: number | null;
    start_time: number | null;
    hold_duration: number;
};

// 2. Functions — take state as first argument
function logout_start(state: Logout, e: Event): void { /* ... */ }
function logout_cancel(state: Logout): void { /* ... */ }
function logout_update(state: Logout): void { /* ... */ }

// 3. Init — creates state, wires listeners, returns state or null
function logout_init(): Logout | null {
    const button = element_get_or_null('logout-button');
    if (!button) return null;

    const state: Logout = { /* ... */ };

    state.button.addEventListener('mousedown', (e) => logout_start(state, e));
    return state;
}
```

Rules:
- State types do NOT have a `State` prefix. The type is just the component name: `Table`, `Logout`, `User_Menu`
- Functions that operate on a component are prefixed with the component name in snake_case
- Init functions return `null` if the component's root element is missing (the component is not on this page)
- Event listeners are attached to the most specific container possible
- Use event delegation on parent elements when child elements are dynamic

### Reusable UI Components

When a piece of UI behavior is needed in multiple contexts (e.g., a dropdown used both as a form input and as a row selector), structure it as:

**Primitive layer** — Generic behavior functions that operate on DOM elements with a known `data-*` attribute structure. They handle mechanics only: positioning, filtering, keyboard navigation, open/close. They do NOT know what happens when the user makes a selection. They report the selection back to the caller.

**Application layer** — Specific code that wires the primitive to a concrete action. Each use case calls the same primitives but provides its own "on select" logic.

```
Primitives:     dropdown_open, dropdown_close, dropdown_filter, dropdown_navigate
                (operate on any [data-dropdown] element)

Use case A:     enum input — on select, set hidden input value
Use case B:     row selector — on select, fetch and insert a new row
```

This separation prevents coupling between unrelated features that share UI mechanics. The primitive is a dumb tool. The caller decides what to do with it.

## Error Handling

**Startup**: Panic if anything prevents the application from being fully initialized. A half-started server is worse than a crash with a clear error.

**Runtime (HTTP handlers)**: Never panic. Return errors, log them, respond with the appropriate HTTP status. Use response helpers (`ServerError`, `Forbidden`, `ClientError`) to keep handlers clean.

**Logging**: Evolving. Currently using `slog` with structured fields. This area needs more work.

## SQL File Conventions

SQL queries live in separate `.sql` files, one query per file.

### Directory structure

- `sql/` — if there is a single database
- `sql_{name}/` — if there are multiple databases (e.g., `sql_master/`, `sql_year/`)
- `sql_both/` — shared migrations or queries used across databases

### File naming

Pattern: `{table}_{operation}_{target}_{filter}.sql`

| Part        | Required | Description                                           | Examples                        |
|-------------|----------|-------------------------------------------------------|---------------------------------|
| `table`     | yes      | Database table name                                   | `b_tabele`, `b_kody__podtabele` |
| `operation` | yes      | SQL verb: `select`, `insert`, `replace`, `update`, `delete`, `count` | `select`            |
| `target`    | yes      | Which columns: `all` for all, or specific column names | `all`, `dane`, `kod_tytul`      |
| `filter`    | no       | `where_` prefix, conditions joined with `_and_`       | `where_podtabela`, `where_idgr_and_podtabela` |

The operation verb is always present. `all` means all columns. Filters always start with `where_`. Join details are omitted — they are implementation, not interface.

Examples:
```
b_tabele_select_all.sql
b_tabele_select_tabela_tytul.sql
b_kolumny_select_all_where_podtabela.sql
b_bdgrobmsp_select_dane_where_idgr_podtabela.sql
b_bdgrobmsp_replace_dane.sql
b_blokady_select_all_where_podtabela_and_kod.sql
b_kody__podtabele_select_kod_tytul_where_podtabela.sql
b_statusy_select_all_where_idbr.sql
```

Always use parameterized queries. Never interpolate values into SQL strings.
