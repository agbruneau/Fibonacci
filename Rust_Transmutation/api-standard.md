# API Standard & Conventions

This guide defines the standard for implementing API endpoints in the `fibcalc` Rust port. It ensures consistency, readability, and maintainability across the codebase.

## 1. Skeleton Structure

Each API module should follow this structure. Handlers are typically defined in `src/server/handlers/` or `src/api/` depending on the project layout.

```rust
// [License Header if applicable]

use axum::{
    extract::{Query, State},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use serde::{Deserialize, Serialize};
use crate::error::AppError; // Centralized error handling
use crate::AppState; // Application state (DB, services, config)

// --- 1. Request/Response DTOs ---

#[derive(Deserialize)]
pub struct MyRequestParams {
    pub n: u64,
}

#[derive(Serialize)]
pub struct MyResponse {
    pub result: String,
    pub duration_ms: u64,
}

// --- 2. Documentation ---

/// Handles the [Action Name] request.
///
/// Detailed description of what this endpoint does.
///
/// # Arguments
///
/// * `State(state)` - Application state (dependency injection).
/// * `Query(params)` - Query parameters (validated via Serde).
///
/// # Returns
///
/// * `Result<impl IntoResponse, AppError>` - JSON response or standardized error.
pub async fn handle_my_action(
    State(state): State<AppState>,
    Query(params): Query<MyRequestParams>,
) -> Result<impl IntoResponse, AppError> {
    // --- 3. Validation (if extra needed beyond types) ---
    if params.n == 0 {
        return Err(AppError::BadRequest("Parameter 'n' must be positive".into()));
    }

    // --- 4. Core Logic Execution ---
    // Delegate complex logic to the service layer, do not write algorithms here.
    let result = state.service.perform_action(params.n).await?;

    // --- 5. Response Construction ---
    let response = MyResponse {
        result: result.value.to_string(),
        duration_ms: result.duration.as_millis() as u64,
    };

    Ok(Json(response))
}
```

## 2. Naming Conventions

| Component            | Convention                     | Example                              |
| :------------------- | :----------------------------- | :----------------------------------- |
| **File Name**        | `snake_case`                   | `health_check.rs`, `calculate.rs`    |
| **Handler Function** | `snake_case`, prefix `handle_` | `handle_health`, `handle_calculate`  |
| **Structs (DTOs)**   | `PascalCase`                   | `CalculateRequest`, `HealthResponse` |
| **Variables**        | `snake_case`                   | `calculation_result`                 |
| **Route Path**       | `kebab-case`                   | `/api/v1/fibonacci-sequence`         |

## 3. Do's and Don'ts

### ✅ Do (Recommended)

- **Do** use `serde` for all input parsing and output serialization.
- **Do** use `axum::extract` types (`State`, `Query`, `Json`) to leverage declarative validation.
- **Do** return `Result<impl IntoResponse, AppError>` to centralize error formatting (e.g., mapping `anyhow::Error` to 500 Internal Server Error).
- **Do** keep handlers thin. They should only parse input, call the service layer, and format the output.
- **Do** document every public handler with RustDoc (`///`).

### ❌ Don't (Avoid)

- **Don't** put business logic (like Fibonacci algorithms) inside the handler function.
- **Don't** use `unwrap()` or `expect()` inside handlers; this causes the server to panic and crash. Always propagate errors.
- **Don't** manually format JSON strings using `format!()`. Use `serde_json`.
- **Don't** use global state (e.g., `static mut`). Use `State` injection.
- **Don't** return raw `String` or `&str` unless it's a simple text endpoint; prefer structured JSON.

## 4. Reference Implementation (The "Golden Standard")

See `src/server/handlers/health.rs` (hypothetical reference) for the simplest complete example of this pattern.

## 5. Integration Testing

Every API handler should have corresponding integration tests. Use `axum::test_helpers` or the `tower::ServiceExt` pattern.

### Test Setup

```rust
// tests/common/mod.rs
use axum::{body::Body, http::Request, Router};
use tower::ServiceExt;

pub async fn create_test_app() -> Router {
    // Initialize app with test configuration
    crate::create_router(TestConfig::default())
}

pub async fn parse_body<T: serde::de::DeserializeOwned>(
    response: axum::response::Response,
) -> T {
    let body = axum::body::to_bytes(response.into_body(), usize::MAX).await.unwrap();
    serde_json::from_slice(&body).unwrap()
}
```

### Example Test

```rust
// tests/api/calculate_test.rs
use axum::{body::Body, http::{Request, StatusCode}};
use tower::ServiceExt;

#[tokio::test]
async fn test_calculate_returns_correct_fibonacci() {
    // Arrange
    let app = create_test_app().await;

    // Act
    let response = app
        .oneshot(
            Request::builder()
                .uri("/calculate?n=10")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    // Assert
    assert_eq!(response.status(), StatusCode::OK);
    let body: CalculateResponse = parse_body(response).await;
    assert_eq!(body.result, "55");
}

#[tokio::test]
async fn test_calculate_rejects_invalid_input() {
    let app = create_test_app().await;

    let response = app
        .oneshot(
            Request::builder()
                .uri("/calculate?n=-1")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::BAD_REQUEST);
}
```
