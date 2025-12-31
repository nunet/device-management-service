{
  description = "NuNet Device Management Service (DMS) - NuNet CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  /*
    Future improvements:
    - gomod2nix: one derivation per dependency + no vendorHash effort
        - https://github.com/nix-community/gomod2nix
        - https://www.tweag.io/blog/2021-03-04-gomod2nix/
  */
  outputs =
    { self
    , nixpkgs
    , flake-utils
    ,
    }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
    in
    flake-utils.lib.eachSystem supportedSystems (
      system:
      let
        pkgs = import nixpkgs { inherit system; };

        # Last commit and build date derived from the flake metadata,
        # meaning the build date is not really the build date but the
        # last commit date.
        # we can't derive the actual DMS version or build date without
        # using impure means
        shortRev = builtins.substring 0 7 (self.rev or "unknown");
        buildDate = toString (self.lastModified or 0); # epoch seconds
        dmsVersion = "unknown+rev-${shortRev}-${buildDate}";

        ldflagsList = [
          "-X gitlab.com/nunet/device-management-service/cmd.Version=${dmsVersion}"
          "-X gitlab.com/nunet/device-management-service/cmd.GoVersion=${pkgs.go.version}"
          "-X gitlab.com/nunet/device-management-service/cmd.BuildDate=${buildDate}"
          "-X gitlab.com/nunet/device-management-service/cmd.Commit=${self.rev or "unknown"}"

          # solves: https://github.com/NVIDIA/nvidia-container-toolkit/issues/49
          "-extldflags=-Wl,-z,lazy"
        ];
      in
      rec {
        packages = rec {
          nunet = pkgs.buildGoModule rec {
            pname = "nunet";
            version = dmsVersion;
            src = ./.;

            subPackages = [ "." ];

            /*
              We have to update the vendorHash everytime go.mod is modified.

              To update it, run:

              `nix run github:Mic92/nix-update -- nunet --flake --version=skip`

              see: https://github.com/Mic92/nix-update
              see 2: https://discourse.nixos.org/t/buildgomodule-how-to-get-vendorsha256/9317

              another method is to run `go mod vendor` and keep the `vendor/` dir
              but it sizes more or less 100MB
            */
            vendorHash = "sha256-eKAGAAHLEcrGxLy5p6Z0tVrz4q8Y8M88ji213Lv6I/E=";

            buildFlags = [ "-buildvcs=false" ];

            # Keep CGO enabled; add systemd for Linux only.
            env = {
              CGO_ENABLED = "1";
            };
            nativeBuildInputs = [ pkgs.pkg-config ];

            buildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux [ pkgs.systemd ];

            ldflags = ldflagsList;

            doCheck = false;

            meta = {
              description = "NuNet Device Management Service (DMS) CLI";
              platforms = supportedSystems;
            };
          };

          default = nunet;
        };

        apps = rec {
          nunet = {
            type = "app";
            program = "${packages.nunet}/bin/device-management-service";
          };
          default = nunet;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            pkg-config
            systemd.dev

            # optional: dev tools
            # addlicense
            # zip
            # gnumake
            # protobuf_27
            # grpcurl
          ];
          shellHook = ''
            echo "NuNet dev shell"
            export CGO_ENABLED=1
            export PKG_CONFIG_PATH="${pkgs.systemd.dev}/lib/pkgconfig:$PKG_CONFIG_PATH"

            # solves: https://github.com/NVIDIA/nvidia-container-toolkit/issues/49
            export GOFLAGS="-ldflags=-extldflags=-Wl,-z,lazy"
          '';
        };
      }
    );
}
