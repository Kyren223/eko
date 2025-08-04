# Self Hosting

## Using NixOS + Flakes

Add eko as a `flake.nix` input:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    eko.url = "github:kyren223/eko/<version>";
  };

  outputs = { nixpkgs, eko, ... }: {
    nixosConfigurations.default = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        eko.nixosModules.eko
        # Your other modules...
      ];
    };
  };
}
```

Replace `<version>` with a release tag, such as `v0.0.0`, or omit it to track
the latest commit on master.

To apply a version change or pull the latest from master, run:

```sh
nix flake update eko

```

Then enable the service via:

```nix
services.eko.enable = true;
services.eko.certFile = "/path/to/certificate";
services.eko.openFirewall = true; # Opens ports 7223 and 443
```

Refer to the [official instance configuration](https://github.com/Kyren223/server/blob/master/nixosModules/eko.nix#L14-L14) for a complete example.

### Notes

- The website (TOS and privacy policy) is served at http://localhost:7443/
- Prometheus metrics are exposed at http://localhost:2112

### Recommended extra steps

- Use [sops-nix](https://github.com/Mic92/sops-nix) to manage secrets like the `certFile`
- Set up **Grafana** for dashboards and visualizations
- Set up **Prometheus** to send metrics to Grafana
- Set up **Loki** and **Grafana Alloy** to ingest logs and send them to Grafana
- Use a reverse proxy (e.g. nginx) to expose the website over HTTPS
- Use [Let's Encrypt](https://letsencrypt.org/) to obtain and renew HTTPS certificates

## Using Docker

Running Eko in Docker (or other container systems) is possible,
but there are no official images yet. Contributions are welcome!

## Standalone

Official standalone instructions are not yet available. Contributions are welcome!

You can refer to [`service.nix`](./service.nix), which defines the systemd service used by the official instance.
While it’s written in Nix, it should be straightforward to adapt into a regular systemd unit.
It also serves as a reference for the flags and environment variables Eko expects.

Note: Eko exposes Prometheus metrics and structured logs by default.
These are optional, and are used with Grafana, Prometheus and Loki.
Logs can still be accessed manually in the logs directory (formatted as JSON).
