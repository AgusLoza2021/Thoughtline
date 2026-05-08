# Event Bus Specification

> Change: `brain-foundation`
> Status: shipped
> Operation: ADDED (new capability — no prior spec exists)

## Purpose

Defines the in-process pub/sub event bus: its event taxonomy, brain-scoped subscriptions,
non-blocking publish policy, concurrency safety, and the storage layer's obligation to emit
events. WebSocket / external transport is explicitly out of scope for this phase.

---

## Requirements

### Requirement: Event Taxonomy

The bus MUST support exactly these typed events in this phase:

| Event | Payload fields (minimum) |
|---|---|
| `MemoryCreated` | `brain_id`, `memory_id`, `timestamp` |
| `MemoryUpdated` | `brain_id`, `memory_id`, `timestamp` |
| `MemoryDeleted` | `brain_id`, `memory_id`, `timestamp` |
| `LinkCreated` | `brain_id`, `link_id`, `from_id`, `to_id`, `relation`, `timestamp` |
| `LinkDeleted` | `brain_id`, `link_id`, `timestamp` |
| `BrainConfigChanged` | `brain_id`, `timestamp` |

Every event MUST carry a `brain_id` and a `timestamp`. Unknown event types MUST NOT be
publishable via the typed API — the type system MUST prevent it at compile time.

#### Scenario: MemoryCreated event carries required fields

- GIVEN memory M is saved to brain B
- WHEN the bus delivers the event to a subscriber of brain B
- THEN the event is of type `MemoryCreated` with `brain_id=B.id`, `memory_id=M.id`, and a non-zero `timestamp`

---

### Requirement: Brain-Scoped Subscriptions

A subscriber MUST specify a `brain_id` at subscription time. The bus MUST deliver only events
whose `brain_id` matches the subscription. A subscriber MUST NOT receive events for a different brain.

#### Scenario: Subscriber receives only its brain's events

- GIVEN subscriber S1 is subscribed to brain A, and subscriber S2 to brain B
- WHEN a memory is saved to brain A
- THEN S1 receives a `MemoryCreated` event; S2 receives nothing

#### Scenario: Subscribing to a non-existent brain is valid

- GIVEN no memories have been saved to brain C
- WHEN a subscriber registers for brain C
- THEN no error is returned; the subscription is held open awaiting future events

---

### Requirement: Non-Blocking Publish Policy

The bus MUST NOT block the publisher when delivering to subscribers. The bus MUST use a
**buffered channel per subscription** with a bounded capacity. If a subscriber's buffer is full
(slow consumer), the bus MUST drop the event for that subscriber and log a warning. The bus
MUST NOT drop events for other subscribers due to one slow subscriber.

The buffer capacity SHOULD be at least 64 events per subscription. The exact capacity MUST be
configurable at bus construction time (not hard-coded in business logic).

#### Scenario: Slow subscriber does not block publisher

- GIVEN a subscriber whose channel buffer is full (capacity exhausted)
- WHEN a new event is published to the same brain
- THEN the publish call returns immediately; the event is dropped for the slow subscriber and logged; other subscribers are unaffected

#### Scenario: Fast subscriber receives all events within buffer

- GIVEN a subscriber with buffer capacity 64 and 10 consecutive events published
- WHEN the subscriber drains its channel
- THEN all 10 events are received in order

---

### Requirement: Concurrent Safety

The bus MUST be safe for concurrent use by multiple goroutines publishing and subscribing
simultaneously. No data race MUST be detectable when running with `-race`.

#### Scenario: 100 concurrent publishers and 10 subscribers do not race

- GIVEN 10 subscribers registered for brain A and 100 goroutines publishing events to brain A
- WHEN all 100 publishes complete concurrently
- THEN the test passes with `go test -race`; no data race is reported

---

### Requirement: Subscription Lifecycle

Closing a subscription MUST stop event delivery to that subscriber. The bus MUST not leak
goroutines when a subscription is closed. After close, the subscriber's channel MUST be
drained and closed.

#### Scenario: Closing a subscription stops delivery

- GIVEN subscriber S is registered and has received 3 events
- WHEN S's subscription is closed
- THEN no further events are delivered to S; the channel is closed; subsequent reads return zero value and false

#### Scenario: Closing a subscription does not affect other subscribers

- GIVEN subscribers S1 and S2 are both subscribed to brain A
- WHEN S1's subscription is closed
- THEN S2 continues to receive events normally

---

### Requirement: Storage Layer Must Emit Events

The `internal/storage` layer MUST emit the corresponding bus event after every successful
database operation:

| Storage operation | Event emitted |
|---|---|
| `Save()` (new memory) | `MemoryCreated` |
| `Save()` / `Update()` (existing memory) | `MemoryUpdated` |
| `Delete()` | `MemoryDeleted` |
| `CreateLink()` | `LinkCreated` |
| `DeleteLink()` | `LinkDeleted` |
| `UpdateBrainConfig()` | `BrainConfigChanged` |

Events MUST NOT be emitted if the DB operation fails (no event on rolled-back transactions).
Events MUST be emitted AFTER commit, not before.

#### Scenario: Saving a new memory emits MemoryCreated

- GIVEN a subscriber S is registered for brain B
- WHEN `storage.Save(brainID=B, memory)` completes successfully
- THEN S receives exactly one `MemoryCreated` event with the new memory's id

#### Scenario: Failed save does not emit event

- GIVEN a subscriber S is registered for brain B
- WHEN `storage.Save()` returns an error (e.g. constraint violation)
- THEN S receives no event

---

### Requirement: No External Transport

The event bus MUST operate in-process only. It MUST NOT expose events over WebSocket, HTTP
SSE, IPC pipes, or any network interface in this phase. Any such transport is explicitly out of scope.

#### Scenario: Bus has no network listener

- GIVEN the application starts with the event bus initialized
- WHEN the process's open ports are inspected
- THEN no new listening port was opened by the event bus
