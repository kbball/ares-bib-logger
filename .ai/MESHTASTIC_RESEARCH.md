# Meshtastic MQTT Research & Findings

Hard-won knowledge about the Meshtastic MQTT gateway integration. Every item here
caused a bug or required debugging to discover — none of it is in the official docs.

---

## MQTT Topic Structure

```
Subscribe (receive from mesh):
  msh/{Region}/{ChannelNum}/e/{ChannelName}/#

Publish (send to mesh via gateway):
  msh/{Region}/{ChannelNum}/e/{ChannelName}/!ffffffff
```

- `Region`: e.g. `US`, `EU`
- `ChannelNum`: the mesh channel number (not the channel index within the gateway)
- `ChannelName`: the human-readable channel name string (e.g. `LongFast`), **not** the
  channel hash that appears in firmware debug logs as `Ch=0x41`
- The publish leaf must be a broadcast address (`!ffffffff`). Using the gateway's own
  node ID as the leaf causes the gateway to treat the message as a self-originated
  uplink and silently ignore it.

---

## ServiceEnvelope Fields

```
ServiceEnvelope {
  packet    MeshPacket  (field 1)
  channel_id string     (field 2) — the channel name string
  gateway_id string     (field 3) — the gateway node ID with "!" prefix, e.g. "!a3b4c5d6"
}
```

**`gateway_id` must be non-empty.** The firmware's `DecodedServiceEnvelope` wrapper sets
the pointer to NULL for empty strings, and `onReceiveProto` rejects envelopes with a
NULL gateway_id — even though the field is optional in the proto schema.

**`gateway_id` must NOT match the gateway's own node ID.** The firmware silently drops
downlinks whose `gateway_id` matches its own ID, treating them as echoes of messages
it originated. Use a sentinel value that is guaranteed to be different (we use `!00000001`).

---

## Self-Echo Filtering (Two Separate Filters Needed)

The broker echoes our own published messages back on the wildcard `#` subscription.
Two separate conditions can match our own echoes, and both must be filtered:

1. **`from` field matches our gateway's node ID** — the broker echoes uplink packets
   where the mesh sender's node ID is our own gateway.
2. **`gateway_id` matches our downlink sentinel** — the broker echoes our own downlink
   acks back to us. We filter these by checking `env.GetGatewayId() == "!"+alertGatewayID`.

Without filter #2, `parseBibs` would extract numbers from "LOGGED: 42" ack text,
causing an infinite loop of re-logging.

---

## MeshPacket Field Ordering (nanopb Incompatibility)

**Do not use the Go protobuf library to encode downlink MeshPackets.** The Go library
places `oneof` fields after higher-numbered regular fields to satisfy the protobuf spec,
but nanopb (used in the Meshtastic firmware) requires fields to appear in strict
field-number order. The mismatch causes nanopb to silently drop or misparse the packet.

We hand-encode all downlink packets using the `pb*Field` helpers. Fields must be written
in ascending field-number order within each message.

`MeshPacket` field order used:
```
1: from        (fixed32)
2: to          (fixed32)   — 0xFFFFFFFF for broadcast
3: channel     (varint)    — channel index; omit if 0 (default channel)
4: decoded     (bytes)     — oneof: the decoded Data sub-message
6: id          (fixed32)   — packet ID; must be non-zero
9: hop_limit   (varint)    — typically 3
15: hop_start  (varint)    — must equal hop_limit for a fresh packet
```

---

## Packet ID

Must be **non-zero**. The firmware silently drops packets with `id = 0`.

---

## hop_start Must Equal hop_limit

For a freshly originated packet, `hop_start` must be set to the same value as
`hop_limit`. If `hop_start` is absent or 0, the firmware may behave unexpectedly.

---

## from Field Filter

The firmware also silently drops downlinks where the `from` field matches the gateway's
own node ID. This means both `from` and `gateway_id` must be non-self values. We use:

```go
const alertFromNodeID uint32 = 1       // from field in MeshPacket
const alertGatewayID = "00000001"      // gateway_id in ServiceEnvelope (without "!" prefix)
```

Both are guaranteed not to collide with a real gateway node ID in normal use.

---

## Encrypted Packets

If the MQTT gateway is configured with channel encryption enabled, uplink packets arrive
with the `Encrypted` oneof set and `Decoded` is nil. We cannot decrypt these — the
server does not have the channel PSK. The gateway must be configured with
**"Decrypt MQTT"** (or equivalent) enabled to send plaintext to the broker.

Log message when this happens: `mqtt: dropping encrypted packet — no channel key`

---

## Portnum Values

| Portnum | Constant | Value |
|---|---|---|
| TEXT_MESSAGE_APP | `meshtastic.PortNum_TEXT_MESSAGE_APP` | 1 |
| NODEINFO_APP | `meshtastic.PortNum_NODEINFO_APP` | 4 |
| POSITION_APP | `meshtastic.PortNum_POSITION_APP` | 3 |

---

## NODEINFO_APP Packet

Sending a `NODEINFO_APP` packet causes all mesh nodes that receive it to display the
logger with a friendly name instead of the default "Meshtastic XXXX".

`User` proto fields (field numbers matter for hand-encoding):
```
1: id         (string) — "!" + 8-char hex node ID, e.g. "!00000001"
2: long_name  (string) — human-readable name, ~20 chars max
3: short_name (string) — screen name, 4 chars max
```

Sent once on adapter connect. The `Data` sub-message uses `portnum = 4` (NODEINFO_APP)
and `payload = marshaled User bytes`.

---

## Channel Index vs Channel Number

These are two different things that are easy to confuse:

- **Channel number** (`MQTT_CHANNEL_NUM`): Used in the MQTT topic path. Set in the
  Meshtastic gateway's MQTT config. Typically `2`.
- **Channel index** (`MQTT_CHANNEL_INDEX`): The position of the channel within the
  gateway's channel list (0–7). Written into `MeshPacket.channel` (field 3) on downlinks.
  The primary channel is index 0; secondary/custom channels are 1+. Check with
  `meshtastic --info` or the Meshtastic app.

---

## Firmware Debug Log Patterns

Useful patterns when reading RAK4631 serial debug output:

- `Ch=0x41` in router logs — this is the **channel hash** (derived from channel name +
  PSK), not the channel index or number.
- `transport = 5` — packet arrived via MQTT (not LoRa)
- `transport = 1` — packet arrived via LoRa radio
- `Portnum=1` — TEXT_MESSAGE_APP
- `fr=0x00000001` — packet from our alertFromNodeID; confirms our downlink was received
- `Heard new node on ch. 1` — firmware received our NodeInfo or first message from our
  virtual node and is updating its database
- `Ignore dupe incoming msg` — LoRa relay heard our MQTT downlink retransmitted over
  the air and is correctly deduplicating it
