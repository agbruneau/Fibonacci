//! FibRust HTTP Server
//!
//! High-performance Fibonacci API server using Axum with LRU caching.
//!
//! Provides a MessagePack-based API for retrieving Fibonacci numbers.
//!
//! # Endpoints
//!
//! - `GET /fib/:n?algo=[adaptive|fd|mx|fft]`
//!   - Returns the nth Fibonacci number encoded in MessagePack.
//!   - Supports algorithm selection via query parameter.
//! - `GET /cache/stats`
//!   - Returns JSON statistics about the LRU cache (hits, misses, ratio).
//!
//! # Caching
//!
//! The server uses an in-memory Least Recently Used (LRU) cache to store computed results.
//! The cache size is configurable via CLI arguments.

use axum::{
    extract::{Path, Query, State},
    response::{Html, IntoResponse},
    routing::get,
    Json, Router,
};
use clap::Parser;
use fibrust_core::{
    fibonacci_adaptive, fibonacci_fast_doubling, fibonacci_fft, fibonacci_parallel, Algorithm,
};
use ibig::UBig;
use lru::LruCache;
use rmp_serde::encode::to_vec_named;
use serde::{Deserialize, Serialize, Serializer};
use std::net::SocketAddr;
use std::num::NonZeroUsize;
use std::sync::{
    atomic::{AtomicU64, Ordering},
    Arc, Mutex,
};

/// Command-line arguments for the server.
#[derive(Parser)]
#[command(name = "fibrust-server", version, about = "FibRust HTTP API Server")]
struct Args {
    /// Port to listen on.
    #[arg(short, long, default_value_t = 3000)]
    port: u16,

    /// LRU cache size (number of entries to cache).
    #[arg(long, default_value_t = 1000)]
    cache_size: usize,
}

/// Query parameters for the /fib endpoint.
#[derive(Clone, Copy, Deserialize)]
struct Params {
    /// Algorithm to use for calculation (default: adaptive).
    #[serde(default)]
    algo: Algorithm,
}

/// Wrapper for `UBig` to implement custom serialization.
///
/// This wrapper allows us to define a custom `Serialize` implementation
/// for `UBig` that outputs raw bytes (MessagePack) instead of the default
/// implementation (which might be less efficient or string-based).
struct BigIntWrapper(UBig);

impl Serialize for BigIntWrapper {
    /// Serializes the large integer as a byte array (little-endian) for MessagePack.
    /// This is more efficient than string serialization for massive numbers.
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        let bytes = self.0.to_le_bytes();
        serializer.serialize_bytes(&bytes)
    }
}

/// Cache key: `(n, algorithm)`
type CacheKey = (u64, Algorithm);
/// Cache value: pre-serialized MessagePack bytes
type CacheValue = Vec<u8>;

/// Shared application state.
///
/// Contains resources shared across all request handlers.
/// Uses `Arc` for shared ownership and `Mutex`/`Atomic` types for thread safety.
#[derive(Clone)]
struct AppState {
    /// Thread-safe LRU cache.
    /// Protected by a Mutex because `LruCache` is not thread-safe.
    cache: Arc<Mutex<LruCache<CacheKey, CacheValue>>>,
    /// Cache hit counter.
    /// Uses lock-free atomic increment for high performance.
    hits: Arc<AtomicU64>,
    /// Cache miss counter.
    /// Uses lock-free atomic increment for high performance.
    misses: Arc<AtomicU64>,
}

/// Handler for getting a Fibonacci number.
///
/// Route: `GET /fib/:n?algo=[adaptive|fd|mx|fft]`
async fn get_fib(
    State(state): State<AppState>,
    Path(n): Path<u64>,
    Query(params): Query<Params>,
) -> impl IntoResponse {
    let algo = params.algo;
    let cache_key = (n, algo);

    // Check cache first
    {
        let mut cache = state.cache.lock().unwrap();
        if let Some(cached_bytes) = cache.get(&cache_key) {
            state.hits.fetch_add(1, Ordering::Relaxed);
            return (
                [(axum::http::header::CONTENT_TYPE, "application/msgpack")],
                cached_bytes.clone(),
            )
                .into_response();
        }
    }

    // Cache miss - compute result
    state.misses.fetch_add(1, Ordering::Relaxed);

    let result = match algo {
        Algorithm::FastDoubling => fibonacci_fast_doubling(n),
        Algorithm::Parallel => fibonacci_parallel(n),
        Algorithm::Fft => fibonacci_fft(n),
        Algorithm::Adaptive => fibonacci_adaptive(n),
    };

    // Serialize result
    let wrapper = BigIntWrapper(result);
    match to_vec_named(&wrapper) {
        Ok(bytes) => {
            // Store in cache
            {
                let mut cache = state.cache.lock().unwrap();
                cache.put(cache_key, bytes.clone());
            }
            (
                [(axum::http::header::CONTENT_TYPE, "application/msgpack")],
                bytes,
            )
                .into_response()
        }
        Err(e) => (
            axum::http::StatusCode::INTERNAL_SERVER_ERROR,
            format!("Serialization error: {}", e),
        )
            .into_response(),
    }
}

/// Statistics about cache usage.
#[derive(Serialize)]
struct CacheStats {
    hits: u64,
    misses: u64,
    hit_ratio: f64,
    cached_entries: usize,
    cache_capacity: usize,
}

/// Handler for cache statistics.
///
/// Route: `GET /cache/stats`
async fn cache_stats(State(state): State<AppState>) -> Json<CacheStats> {
    let hits = state.hits.load(Ordering::Relaxed);
    let misses = state.misses.load(Ordering::Relaxed);
    let total = hits + misses;
    let hit_ratio = if total > 0 {
        hits as f64 / total as f64
    } else {
        0.0
    };

    let cache = state.cache.lock().unwrap();

    Json(CacheStats {
        hits,
        misses,
        hit_ratio,
        cached_entries: cache.len(),
        cache_capacity: cache.cap().into(),
    })
}

/// Handler for the root path.
///
/// Route: `GET /`
async fn root() -> Html<&'static str> {
    Html(include_str!("index.html"))
}

/// Creates the Axum router with all routes configured.
///
/// This function is separated from `main` to enable integration testing
/// without requiring a live server.
///
/// # Arguments
/// * `cache_size` - Maximum number of entries in the LRU cache.
///
/// # Returns
/// A configured `Router` with all endpoints and shared state.
pub fn create_app(cache_size: usize) -> Router {
    let cache_size = NonZeroUsize::new(cache_size).unwrap_or(NonZeroUsize::new(1000).unwrap());
    let state = AppState {
        cache: Arc::new(Mutex::new(LruCache::new(cache_size))),
        hits: Arc::new(AtomicU64::new(0)),
        misses: Arc::new(AtomicU64::new(0)),
    };

    Router::new()
        .route("/", get(root))
        .route("/fib/{n}", get(get_fib))
        .route("/cache/stats", get(cache_stats))
        .with_state(state)
}

/// Main server entry point.
///
/// Parses CLI arguments, initializes the system, and starts the HTTP server.
pub async fn run() -> anyhow::Result<()> {
    let args = Args::parse();

    println!("FibRust Server v{}", env!("CARGO_PKG_VERSION"));
    println!("Initializing system...");
    fibrust_core::prewarm_system();

    println!("LRU Cache: {} entries", args.cache_size);

    let app = create_app(args.cache_size);

    let addr = SocketAddr::from(([0, 0, 0, 0], args.port));
    println!("Listening on http://{}", addr);

    let listener = tokio::net::TcpListener::bind(addr)
        .await
        .map_err(|e| anyhow::anyhow!("Failed to bind to port {}: {}", args.port, e))?;

    axum::serve(listener, app)
        .await
        .map_err(|e| anyhow::anyhow!("Server error: {}", e))?;

    Ok(())
}
