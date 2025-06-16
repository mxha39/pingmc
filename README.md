# pingmc



## Usage

```bash
./pingmc <host> [flags]
```

### Examples

```bash
# Basic usage
./pingmc example.com

# Force IPv6
./pingmc example.com -6

# Send 'minecraft.net' instead of 'example.com' in the handshake packet
./pingmc example.com -s minecraft.net

---

## Flags

| Flag         | Alias | Description                               |
| ------------ | ----- | ----------------------------------------- |
| `--pp2`      |       | Send Proxy Protocol v2 header             |
| `--spoof`    | `-s`  | Send custom hostname in handshake packet  |
| `--version`  | `-v`  | Minecraft protocol version (default: 766) |
| `--saveIcon` | `-i`  | Save favicon as PNG                       |
| `--show`     |       | Show sample player names                  |
| `--useV4`    | `-4`  | Force IPv4 only                           |
| `--useV6`    | `-6`  | Force IPv6 only                           |

```
## Output Example

```text
──────────────────────────────────────────────────────
Target: play.example.org
IP-Address: 10.0.1.1
Version: Paper 1.21.4 - 769
Players: 2 / 8
Ping: 1 ms

A Minecraft Server

Players:
- Steve (8667ba71b85a4004af54457a9734eed7)
- Alex (ec561538f3fd461daff5086b22154bce)
──────────────────────────────────────────────────────
```