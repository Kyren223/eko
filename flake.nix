{
  description = "Localias is a tool for developers to securely manage local aliases for development servers.";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs";

    flake-utils.url = "github:numtide/flake-utils";

    flake-compat.url = "github:edolstra/flake-compat";
    flake-compat.flake = false;

    nix-filter.url = "github:numtide/nix-filter";
  };

  outputs =
    { self, ... }@inputs:
    inputs.flake-utils.lib.eachDefaultSystem (
      system:
      let
        overlays = [ ];
        pkgs = import inputs.nixpkgs {
          inherit system overlays;
        };
        version = (builtins.readFile ./VERSION);
        buildDate = toString self.lastModified;
        commit = if (builtins.hasAttr "rev" self) then (builtins.substring 0 7 self.rev) else "unknown";
        # vendorHash = pkgs.lib.fakeHash;
        vendorHash = "sha256-dLhLFyrufv3dNlAw1QLlf9/LsHMcUaD9F2byKlC+35E=";
        src =
          let
            # Set this to `true` in order to show all of the source files
            # that will be included in the module build.
            debug-tracing = false;
            source-files = inputs.nix-filter.lib.filter {
              root = ./.;
            };
          in
          (if (debug-tracing) then pkgs.lib.sources.trace source-files else source-files);
        ldflags = [
          "-X github.com/kyren223/eko/embeds.Version=${version}"
          "-X github.com/kyren223/eko/embeds.Commit=${commit}"
          "-X github.com/kyren223/eko/embeds.BuildDate=${buildDate}"
        ];
      in
      rec {
        packages = rec {
          eko = pkgs.buildGoModule {
            pname = "eko";
            version = version;
            vendorHash = vendorHash;
            src = src;
            buildInputs = [ pkgs.go-tools pkgs.gosec ];
            ldflags = ldflags;
            modRoot = "./.";
            subPackages = [ "cmd/client" ];
            doCheck = false;
            preBuild = ''
              export HOME=$(mktemp -d) # For staticheck

              echo "running ci..."

              test -z "$(go fmt ./...)"
              echo "formatting passed"

              ${pkgs.go-tools}/bin/staticcheck ./...
              echo "static analysis passed"

              go test --cover ./...
              echo "tests passed"

              ${pkgs.gosec}/bin/gosec ./...
              echo "gosec passed"

              echo "done"
            '';
            postInstall = ''
              mv $out/bin/client $out/bin/eko
            '';
          };
          eko-server = pkgs.buildGoModule {
            pname = "eko-server";
            version = version;
            vendorHash = vendorHash;
            src = src;
            buildInputs = [ pkgs.goose pkgs.go-tools pkgs.gosec ];
            ldflags = ldflags;
            modRoot = "./.";
            subPackages = [ "cmd/server" ];
            doCheck = false;
            preBuild = ''
              export HOME=$(mktemp -d) # For staticheck

              echo "running ci..."

              test -z "$(go fmt ./...)"
              echo "formatting passed"

              ${pkgs.go-tools}/bin/staticcheck ./...
              echo "static analysis passed"

              go test --cover ./...
              echo "tests passed"

              ${pkgs.gosec}/bin/gosec ./...
              echo "gosec passed"

              echo "running migrations..."
              ${pkgs.goose}/bin/goose fix -dir internal/server/api/migrations
              echo "migrations passed"

              echo "done"
            '';
            postInstall = ''
              mv $out/bin/server $out/bin/eko-server
            '';
          };
          default = eko;
        };

        apps = rec {
          eko = {
            type = "app";
            program = "${packages.eko}/bin/eko";
            meta = {
              description = "A terminal-native social media platform (client)";
              homepage = "https://github.com/kyren223/eko";
              license = pkgs.lib.licenses.agpl3Plus;
              maintainers = with pkgs.lib.maintainers; [ kyren223 ];
              platforms = pkgs.lib.platforms.all;
            };
          };
          eko-server = {
            type = "service";
            program = "${packages.eko}/bin/eko-server";
            meta = {
              description = "A terminal-native social media platform (server)";
              homepage = "https://github.com/kyren223/eko";
              license = pkgs.lib.licenses.agpl3Plus;
              maintainers = with pkgs.lib.maintainers; [ kyren223 ];
              platforms = pkgs.lib.platforms.all;
            };
          };
          default = eko;
        };

      }
    )
    // {
      nixosModules = rec {
        default = eko;
        eko = import ./service.nix inputs;
      };
    };
}
