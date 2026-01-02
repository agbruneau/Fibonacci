//! FibRust HTTP Server - Binary Entry Point
//!
//! This is the main entry point for the fibrust-server binary.
//! The core implementation is in the library crate.

#[path = "main_impl.rs"]
mod main_impl;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    main_impl::run().await
}
