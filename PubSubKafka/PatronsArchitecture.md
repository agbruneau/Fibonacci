# 🏗️ Architecture & Design Patterns

This document details the architectural models and design choices implemented in this project.

## 🧩 Architecture Patterns

### 1. Event-Driven Architecture (EDA)

Induces total decoupling between components via asynchronous messaging.

- **Implementation**: Apache Kafka serves as the central event bus.
- **Benefit**: High availability, horizontal scalability, and simplified extensibility.

### 2. Event Carried State Transfer (ECST)

Each message is "self-contained," carrying the full state required for downstream processing.

- **Benefit**: Eliminates synchronous API calls to upstream services or databases, enhancing consumer autonomy.
- **Resource**: [order.go](file:///c:/Users/agbru/OneDrive/Documents/GitHub/PubSubKafka/order.go) defines the enriched data structure.

### 3. Dual-Stream Logging (Audit vs. Health)

Clear separation of concerns regarding system journaling.

- **Service Health Monitoring** (`tracker.log`): Technical metrics and system lifecycle events.
- **Business Audit Trail** (`tracker.events`): An immutable, high-fidelity journal of all business event flows.

### 4. Graceful Shutdown

Services intercept system interrupt signals (`SIGINT` / `SIGTERM`).

- **Mechanics**: Ensures Kafka buffers are flushed and file descriptors are safely closed before termination, preventing data loss.

## 🛠️ Infrastructure & DevOps

- **Kafka KRaft Mode**: Utilizes the modern KRaft protocol, removing the dependency on Zookeeper for a leaner infrastructure.
- **Go Build Tags**: Orchestrates multiple entry points and conditional logic via compilation tags (`producer`, `tracker`, `monitor`).
- **Automated Lifecycle**: [start.sh](file:///c:/Users/agbru/OneDrive/Documents/GitHub/PubSubKafka/start.sh) and [stop.sh](file:///c:/Users/agbru/OneDrive/Documents/GitHub/PubSubKafka/stop.sh) provide reliable, environment-aware orchestration.
