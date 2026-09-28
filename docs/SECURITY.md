# Security Model

SocketLens is intended for systems and networks you own or are authorized to inspect.

The initial release deliberately reduces scan scope:

- only loopback and private IP addresses are accepted
- URLs and filesystem paths are rejected
- each scan is capped at 1024 unique TCP ports
- connection timeouts are bounded
- scan execution has an overall deadline
- worker concurrency is bounded
- no raw packets are used
- no credential guessing or authentication attempts are implemented
- no exploit or vulnerability execution is implemented

These restrictions are part of the product design, not only UI guidance.
