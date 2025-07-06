# Eko Protocol V2

## Packet Structure

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|    Version    |En.|    Type   |         Payload Length        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|              Payload... Payload Length bytes ...              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

Order of bytes is from left to right, top to bottom.
The first byte is always the version, any bytes after it
depend on the specific value of the version byte.

- Encoding: 0-3, determines the way the payload was encoded
  - 0: JSON
  - 1: MsgPack
  - 2: Reserved for future use
  - 3: Reserved for future use
- Type: 0-63, determines the type ("schema"), of the payload
- Payload Length: 0-65531, determines how long the payload is in bytes
- Payload: 0 to 65531 bytes long, depending on the payload size (~64kb)

## Handshake

The first time a connection is established, the following packets are exchanged.

- Server sends a TosInfo (Type 1) with the Terms of Service and Privacy Policy
- Client must respond with an AcceptTos (Type 2) with an agree boolean of true
  - If the client sends any other type of packet, or the boolean is false, the server may close the connection

After the handshake, the client may send any unauthenticated packets.
And the server may stream any additional packets.

### Authentication

At any time, while authenticated or not, the client may request a nonce (GetNonce, Type 3).
The server then must responsd with NonceInfo (Type 4).

It's recommended for the server to use a cryptographically secure random number generator,
and to rotate the value every minute.

To authenticate, a client must send an Authenticate request (Type 5),
the request must include a public key, and a signature (64 bytes) of the nonce.

The server must then respond in one of the following ways:

- With an Error (Type 0) with any message, indicating failed authentication
- With a UsersInfo (Type 6) indicating success
  - Must include exactly one User (corresponding to the authenticated user)

Note: if the client takes too long between the nonce request, the nonce may have been rotated
and the client will need to redo these steps.

## Error handling

The server may close a connection only in these cases:

- the client has already closed the connection
- the server has finished processing and sending all reqeusts
- the server experienced an abrupt termination (SIGKILL, powerloss, etc)
- after sending a TosInfo and a receiving any packet except of AcceptTos with a value of true

The server may stop receiving data (and drop any partial requests) at any time (in response to a SIGINT/SIGTERM).

The client may close a connection at any time but data loss may occur.

### Malformed Packets

- unsupported/invalid version: connection can be closed immediately
- unsupported encoding: server must respond with an error type, may use any encoding, client may close the connection
- unknown type: server must respond with an error, client may close the connection
- malformed paylod: server must respond with an error, client may close the connection

For application errors such as a client asking to send a message in a non-existent Frequency,
the server must respond with an error packet.
For internal errors such as database failure, the server must respond, it may choose to
disclose as much information as it wants, or just say "internal server error".
