# Terminology

TODO

## TODO

ID - A snowflake ID, a snowflake ID is a unique identifier used across the client and server, it embeds metadata including the date it was generated which can be retrieved for display purposes.

Receiver ID - usually refers to the "other" person's ID in a 1:1 messaging, but can also refer to the literal `receiver_id` SQL field on the `messages` table which may be either the "other" person, or the user itself.

Frequency ID

Chat ID - A snowflake ID that refers to either a frequency ID or a receiver ID

Signal - A currently open chat between 2 individual users ("DM")
