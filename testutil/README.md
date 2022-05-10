# Testing


```yaml
---
scenarios:
  "client requests to create a session when one does not exist":
    given: []
    when:
      create
        principal:
          email: ?em

    then:
      created:
        principal:
          email: ?em

  "client requests a session when one already exists for the same user":
    given:
      - created
          principal:
            email: ?em
    when:
      create:
        principal:
          email: ?em

    error: session-already-exists

  "client requests a session when a different session exists":
    given:
      - created
          id: ?id1
          principal:
            email: ?em1
    when:
      create:
        principal:
          email: ?em2
    then:
      created:
        id: ?id2
        principal:
          email: ?em2

  "client requests to revoke a session that exists":
    given:
      - created:
          id: ?id
    when:
      revoke:
        id: ?id
    then:
      revoked:
        id: ?id

  "client requests to extend a session":
    config:
      ttl: 20m
    given:
      - created:
          id: ?id
          expiry: ?ex
    when:
      extend:
        id: ?id
    then:
      extended:
        id: ?id
        ttl: ?ex + ?ttl

```

Requirements

- type registry to map type names to values
- support errors in registry
- generate valid required values for commands and events
- map (nested) fields in a value to references
- ensure reference values are the same


Operators:

- assert different values
- add durations to times
- add numbers
