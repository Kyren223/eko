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
        buildDate = builtins.readFile (
          pkgs.runCommand "build-date" { } ''
            ${pkgs.coreutils}/bin/date --date=@${toString self.lastModified} +%Y-%m-%d -u > $out
          ''
        );
        # buildDate = builtins.readFile (
        #   pkgs.runCommand "build-date" { } ''
        #     date -u +'%Y-%m-%d' > $out
        #   ''
        # );
        commit = if (builtins.hasAttr "rev" self) then (builtins.substring 0 7 self.rev) else "unknown";
        # vendorHash = pkgs.lib.fakeHash;
        vendorHash = "sha256-2yCQ40T5N90lKpPOc+i6vz+1mI/p4Ey6PdRCJbGD+TE=";
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
            buildInputs = [ ];
            ldflags = ldflags;
            modRoot = "./.";
            subPackages = [ "cmd/client" ];
            doCheck = false;
            postInstall = ''
              mv $out/bin/client $out/bin/eko
            '';
          };
          eko-server = pkgs.buildGoModule {
            pname = "eko-server";
            version = version;
            vendorHash = vendorHash;
            src = src;
            buildInputs = [ ];
            ldflags = ldflags;
            modRoot = "./.";
            subPackages = [ "cmd/server" ];
            doCheck = false;
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

        nixosModules = rec {
          default = eko;
          eko = import ./service.nix inputs;
        };

        # TODO: make my own devshell?
        # devShells = rec {
        #   default = pkgs.mkShell {
        #     packages = with pkgs;
        #       [
        #         # golang
        #         go
        #         delve
        #         go-outline
        #         golangci-lint
        #         gopkgs
        #         gopls
        #         gotools
        #         # nix
        #         nixpkgs-fmt
        #         # other tools
        #         just
        #         cobra-cli
        #       ];
        #
        #     shellHook = ''
        #       # The path to this repository
        #       shell_nix="''${IN_LORRI_SHELL:-$(pwd)/shell.nix}"
        #       workspace_root=$(dirname "$shell_nix")
        #       export WORKSPACE_ROOT="$workspace_root"
        #
        #       # We put the $GOPATH/$GOCACHE/$GOENV in $TOOLCHAIN_ROOT,
        #       # and ensure that the GOPATH's bin dir is on our PATH so tools
        #       # can be installed with `go install`.
        #       #
        #       # Any tools installed explicitly with `go install` will take precedence
        #       # over versions installed by Nix due to the ordering here.
        #       export TOOLCHAIN_ROOT="$workspace_root/.toolchain"
        #       export GOROOT=
        #       export GOCACHE="$TOOLCHAIN_ROOT/go/cache"
        #       export GOENV="$TOOLCHAIN_ROOT/go/env"
        #       export GOPATH="$TOOLCHAIN_ROOT/go/path"
        #       export GOMODCACHE="$GOPATH/pkg/mod"
        #       export PATH=$(go env GOPATH)/bin:$PATH
        #       export CGO_ENABLED=1
        #
        #       # Make it easy to test while developing; add the golang and nix
        #       # build outputs to the path.
        #       export PATH="$workspace_root/bin:$workspace_root/result/bin:$PATH"
        #     '';
        #
        #     # Need to disable fortify hardening because GCC is not built with -oO,
        #     # which means that if CGO_ENABLED=1 (which it is by default) then the golang
        #     # debugger fails.
        #     # see https://github.com/NixOS/nixpkgs/pull/12895/files
        #     hardeningDisable = [ "fortify" ];
        #   };
        # };
      }
    );
}
