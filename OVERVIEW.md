### `sdk-entities` repository

#### Project Overview

This repository is a component of the Slidebolt SDK that provides concrete implementations for common smart home device domains, such as `light` and `switch`. It builds upon the foundational types defined in the `sdk-types` package.

#### Architecture

The `sdk-entities` package provides a structured, type-safe way for plugin developers to work with different kinds of devices. For each supported domain, this package offers:

-   **Domain-Specific Types**: It defines the specific Go structs for a domain's `State`, `Command`, and `Event` payloads.
-   **Domain Registration**: Each domain package contains an `init()` function that calls `types.RegisterDomain()`, automatically registering its capabilities (supported commands and their parameters) with the global domain registry. This allows the entire system to be aware of what a `light` or a `switch` can do.
-   **State Management Helper (`Store`)**: A key feature is the `Store` object. A developer can `Bind()` a generic `types.Entity` to a domain-specific store (e.g., `light.Bind(entity)`) to get a type-safe API for manipulating its state. This abstracts away the need to manually handle JSON and provides convenient methods like `SetBrightness(50)` or `TurnOn()`.

By providing these helpers, this package greatly simplifies the process of creating plugins that interact with common device types.

#### Key Files

| File | Description |
| :--- | :--- |
| `go.mod` | Defines the Go module and its dependency on `sdk-types`. |
| `light/light.go` | Contains the complete implementation for the `light` domain, including its data structures, domain description, and the `Store` helper. |
| `switch/switch.go` | Contains the complete implementation for the `switch` domain, with its own data structures and `Store` helper. |

#### Available Commands

This is a library package and is not intended to be run directly. It is imported and used by other Slidebolt plugins to interact with entities in a standardized way.
