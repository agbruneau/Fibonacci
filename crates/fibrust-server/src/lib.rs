//! FibRust HTTP Server Library
//!
//! This module exposes the core server functionality for testing purposes.
//!
//! The main entry point is [`create_app`], which creates a configured Axum router.

#[path = "main_impl.rs"]
mod main_impl;

pub use main_impl::create_app;
