# Sample Trigger Kit

Reference Emerald trigger plugin bundle implemented with the Go SDK.

It includes:

- `trigger:plugin/sample-trigger-kit/heartbeat`

The trigger node demonstrates:

- plugin-defined trigger registration in the editor
- config validation through the normal plugin validation path
- the long-lived trigger runtime stream that receives full subscription snapshots
- emitted trigger events that start the exact subscribed root node in Emerald

## Build

From the repository root:

```powershell
go build -o .\examples\plugins\sample-trigger-kit\bin\sample-trigger-kit.exe .\examples\plugins\sample-trigger-kit
```

If you are building on a non-Windows host, update `plugin.json` to point at the binary name you produce.

## Try It

Copy the whole `sample-trigger-kit` directory under your configured plugin root, for example:

```text
.agents/plugins/sample-trigger-kit/
```

Then start Emerald or use `Settings -> Rediscover Plugins`.

Create a pipeline with the `Heartbeat Trigger` node and connect it to a `Return` node or a message action. When the pipeline is active, the plugin receives the full active subscription snapshot and emits periodic events with the configured message, sequence, and timestamp payload.
