//! Integration tests for the FibRust HTTP Server.
//!
//! These tests verify the API endpoints by making HTTP requests
//! to the server without starting a live network listener.

use axum::{
    body::Body,
    http::{Request, StatusCode},
};
use http_body_util::BodyExt;
use tower::ServiceExt;

// Import the create_app function from the server binary.
// Note: This requires the function to be `pub` in main.rs.
use fibrust_server::create_app;

/// Helper to create a test app with a small cache.
fn test_app() -> axum::Router {
    create_app(100)
}

// ============================================================================
// Basic Fibonacci Endpoint Tests
// ============================================================================

#[tokio::test]
async fn get_root_returns_html() {
    let app = test_app();

    let response = app
        .oneshot(Request::get("/").body(Body::empty()).unwrap())
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    assert_eq!(
        response.headers().get("content-type").unwrap(),
        "text/html; charset=utf-8"
    );

    let body = response.into_body().collect().await.unwrap().to_bytes();
    let body_str = String::from_utf8(body.to_vec()).unwrap();
    assert!(body_str.contains("FibRust API"));
}

#[tokio::test]
async fn get_fib_0_returns_success() {
    let app = test_app();

    let response = app
        .oneshot(Request::get("/fib/0").body(Body::empty()).unwrap())
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    assert_eq!(
        response.headers().get("content-type").unwrap(),
        "application/msgpack"
    );
}

#[tokio::test]
async fn get_fib_1_returns_success() {
    let app = test_app();

    let response = app
        .oneshot(Request::get("/fib/1").body(Body::empty()).unwrap())
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
}

#[tokio::test]
async fn get_fib_10_returns_success() {
    let app = test_app();

    let response = app
        .oneshot(Request::get("/fib/10").body(Body::empty()).unwrap())
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);

    // Verify we get MessagePack bytes
    let body = response.into_body().collect().await.unwrap().to_bytes();
    assert!(!body.is_empty(), "Response body should not be empty");
}

#[tokio::test]
async fn get_fib_large_value() {
    let app = test_app();

    let response = app
        .oneshot(Request::get("/fib/1000").body(Body::empty()).unwrap())
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
}

// ============================================================================
// Algorithm Selection Tests
// ============================================================================

#[tokio::test]
async fn get_fib_with_fd_algorithm() {
    let app = test_app();

    let response = app
        .oneshot(
            Request::get("/fib/100?algo=fd")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
}

#[tokio::test]
async fn get_fib_with_par_algorithm() {
    let app = test_app();

    let response = app
        .oneshot(
            Request::get("/fib/100?algo=par")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
}

#[tokio::test]
async fn get_fib_with_mx_alias() {
    let app = test_app();

    // "mx" is an alias for "par" (parallel)
    let response = app
        .oneshot(
            Request::get("/fib/100?algo=mx")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
}

#[tokio::test]
async fn get_fib_with_fft_algorithm() {
    let app = test_app();

    let response = app
        .oneshot(
            Request::get("/fib/100?algo=fft")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
}

#[tokio::test]
async fn get_fib_with_adaptive_algorithm() {
    let app = test_app();

    let response = app
        .oneshot(
            Request::get("/fib/100?algo=adaptive")
                .body(Body::empty())
                .unwrap(),
        )
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
}

// ============================================================================
// Cache Statistics Endpoint Tests
// ============================================================================

#[tokio::test]
async fn cache_stats_returns_json() {
    let app = test_app();

    let response = app
        .oneshot(Request::get("/cache/stats").body(Body::empty()).unwrap())
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::OK);
    assert_eq!(
        response.headers().get("content-type").unwrap(),
        "application/json"
    );

    let body = response.into_body().collect().await.unwrap().to_bytes();
    let stats: serde_json::Value = serde_json::from_slice(&body).unwrap();

    // Verify expected fields exist
    assert!(stats.get("hits").is_some());
    assert!(stats.get("misses").is_some());
    assert!(stats.get("hit_ratio").is_some());
    assert!(stats.get("cached_entries").is_some());
    assert!(stats.get("cache_capacity").is_some());
}

#[tokio::test]
async fn cache_stats_initial_values() {
    let app = test_app();

    let response = app
        .oneshot(Request::get("/cache/stats").body(Body::empty()).unwrap())
        .await
        .unwrap();

    let body = response.into_body().collect().await.unwrap().to_bytes();
    let stats: serde_json::Value = serde_json::from_slice(&body).unwrap();

    // Fresh app should have 0 hits and 0 misses
    assert_eq!(stats["hits"], 0);
    assert_eq!(stats["misses"], 0);
    assert_eq!(stats["hit_ratio"], 0.0);
    assert_eq!(stats["cached_entries"], 0);
    assert_eq!(stats["cache_capacity"], 100);
}

// ============================================================================
// Invalid Route Tests
// ============================================================================

#[tokio::test]
async fn invalid_route_returns_404() {
    let app = test_app();

    let response = app
        .oneshot(Request::get("/invalid/route").body(Body::empty()).unwrap())
        .await
        .unwrap();

    assert_eq!(response.status(), StatusCode::NOT_FOUND);
}

// ============================================================================
// Consistency Tests
// ============================================================================

#[tokio::test]
async fn different_algorithms_produce_response() {
    // All algorithms should produce a valid response for the same input
    for algo in ["fd", "par", "fft", "adaptive"] {
        let app = test_app();
        let uri = format!("/fib/50?algo={}", algo);

        let response = app
            .oneshot(Request::get(&uri).body(Body::empty()).unwrap())
            .await
            .unwrap();

        assert_eq!(
            response.status(),
            StatusCode::OK,
            "Algorithm {} should return OK",
            algo
        );
    }
}
