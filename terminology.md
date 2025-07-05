# Terminology

TODO

## TODO

ID - A snowflake ID, a snowflake ID is a unique identifier used across the client and server, it embeds metadata including the date it was generated which can be retrieved for display purposes.

User ID - A snowflake ID for a user, used as the primary way to refer to a user in a unique way (like opening a signal with a user)

Receiver ID - usually refers to the "other" person's ID in a 1:1 messaging, but can also refer to the literal `receiver_id` SQL field on the `messages` table which may be either the "other" person, or the user itself.

Frequency ID

Chat ID - A snowflake ID that refers to either a frequency ID or a receiver ID

Signal - A currently open chat between 2 individual users ("DM")

Session - A struct that helps with managing a single active connection, it contains the addr of the connection and has methods for reading/writing to the connection

Unauthenticated Session - A session that has not yet been authenticated, so it doesn't have a User ID or public key yet, and has not been added to the server sessions map

Authenticated Session - A session that is authenticated and attached to a user, it has a UserID and a public key and it has been mapped into the server's session map based on it's user ID.
