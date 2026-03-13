# Package Level Requirements

Tests for the sdk-entities project should verify:

- **Domain Validation**: Entities must conform to their defined domains (light, switch, sensor, etc.).
- **Action Schema**: Entity actions must follow the expected payload formats.
- **State Normalization**: Reported/Desired state must be properly normalized and validated.
