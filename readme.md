# 🚀 Kafka Order Tracking System

[![Go Version](https://img.shields.io/badge/Go-1.22.0-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Kafka](https://img.shields.io/badge/Apache_Kafka-3.7.0-white?style=flat&logo=apache-kafka)](https://kafka.apache.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A robust, enterprise-grade **Event-Driven Architecture (EDA)** demonstration using **Go** and **Apache Kafka**. This project simulates a complete e-commerce order lifecycle—from generation to real-time tracking—featuring high observability via a dedicated Terminal User Interface (TUI).

---

## 🏗 System Architecture

The ecosystem consists of three decoupled core services:

1.  **📦 Producer (`producer`)**: Simulates customer activity by generating enriched order events and streaming them to the `orders` Kafka topic.
2.  **⚙️ Tracker (`tracker`)**: Consumes order events in real-time, performing validation and maintaining a comprehensive audit trail.
3.  **📊 Monitor (`log_monitor`)**: A sophisticated TUI dashboard providing live visualization of system performance, throughput, and success rates.

For a deep dive into the design patterns used, see [PatronsArchitecture.md](file:///c:/Users/agbru/OneDrive/Documents/GitHub/PubSubKafka/PatronsArchitecture.md).

---

## 🌟 Key Features & Design Patterns

- **Event-Driven Architecture (EDA)**: Complete decoupling of services through asynchronous messaging.
- **Event Carried State Transfer (ECST)**: Self-contained messages that include all necessary context.
- **Guaranteed Delivery**: Implements Kafka delivery reports (ACKs) to ensure data integrity.
- **Dual-Stream Observability**: Technical health (`tracker.log`) vs Business Audit (`tracker.events`).
- **Graceful Shutdown**: Strict handling of `SIGTERM`/`SIGINT` for zero-data-loss termination.

---

## 🛠 Prerequisites

Ensure the following are installed:

1.  **Docker** and **Docker Compose** (V2).
2.  **Go** (version 1.22.0 or higher).
3.  **Make** (optional, but highly recommended for CLI efficiency).
4.  An **ANSI-compatible terminal** (for the TUI monitor).

---

## ⌨️ Command Line Interface (Makefile)

The project includes a comprehensive `Makefile` to simplify common operations.

| Command           | Description                                                          |
| :---------------- | :------------------------------------------------------------------- |
| `make build`      | Compile all service binaries (`producer`, `tracker`, `log_monitor`). |
| `make run`        | Deploy Kafka and start all background services (Linux/macOS).        |
| `make stop`       | Gracefully shut down all services and infrastructure.                |
| `make test`       | Run the complete test suite.                                         |
| `make test-cover` | Run tests and generate an HTML coverage report.                      |
| `make docker-up`  | Start only the Kafka infrastructure.                                 |
| `make clean`      | Remove all binaries and log files.                                   |
| `make help`       | Display all available commands.                                      |

---

## 🚀 Getting Started

### 1. Automated Deployment (Linux/macOS)

```bash
make run
```

This script handles Kafka health checks, topic creation, and background service initialization.

### 2. Manual Execution (All Platforms)

If you prefer manual control or are on Windows:

```bash
# 1. Start Kafka
make docker-up

# 2. Launch Services in separate terminals
go run -tags kafka cmd/producer/main.go
go run -tags kafka cmd/tracker/main.go
```

---

## 📊 Monitoring

Launch the TUI monitor for real-time visualization:

```bash
make run-monitor
```

- **Controls**: Press `q` to exit.
- **Insights**: Monitor msg/sec, success rates, and live logs.

---

## 🧪 Build Tags & Testing

This project uses Go **Build Tags** for modular compilation:

| Tag        | Purpose                                      |
| :--------- | :------------------------------------------- |
| `producer` | Includes producer-specific logic.            |
| `tracker`  | Includes consumer/tracker-specific logic.    |
| `monitor`  | Includes terminal UI dependencies and logic. |
| `kafka`    | Includes Kafka client initialization.        |

### Running Tests

```bash
# All tests
make test

# Coverage report
make test-cover
```

---

## 🗺 Future Roadmap

We are evolving this demo into a production-ready template. Detailed improvements can be found in [amelioration.md](file:///c:/Users/agbru/OneDrive/Documents/GitHub/PubSubKafka/amelioration.md).

- [x] **1. Architecture**: Migrate to Standard Go Package Structure (`/cmd`, `/internal`, `/pkg`).
- [ ] **2. Configuration**: Implementation of external configuration (`config.yaml`).
- [ ] **3. Resilience**: Add Retry Patterns with Exponential Backoff and Dead Letter Queues (DLQ).
- [ ] **4. CI/CD**: Integrate GitHub Actions for automated testing and linting.
- [ ] **5. Observability**: Export Prometheus metrics and OpenTelemetry traces.

---

## 📂 Project Structure

- **`cmd/`**: Application entry points.
  - `producer/`: Order generation service.
  - `tracker/`: Consumer and validation service.
  - `monitor/`: TUI dashboard service.
- **`pkg/`**: Public libraries and shared logic.
  - `models/`: Shared domain entities (Order, CustomerInfo).
  - `producer/`: Kafka producer implementation.
  - `tracker/`: Kafka consumer and observability logic.
  - `monitor/`: TUI rendering and log parsing logic.
- **`Makefile`**: Operational orchestration.
- **`docker-compose.yaml`**: Infrastructure as code.
- **`*.md`**: Documentation and Roadmap.
