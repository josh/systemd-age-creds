{
  description = "systemd-age-creds";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
  };

  outputs =
    { self, nixpkgs }:
    let
      inherit (nixpkgs) lib;
      mapListToAttrs = f: list: builtins.listToAttrs (builtins.map f list);
      callPackage = {
        aarch64-linux = nixpkgs.legacyPackages.aarch64-linux.callPackage;
        x86_64-linux = nixpkgs.legacyPackages.x86_64-linux.callPackage;
      };
      checkMatrix = import ./nix/tests/nixos-system-unit-matrix.nix { inherit lib; };
    in
    {
      packages = {
        aarch64-linux.systemd-age-creds = callPackage.aarch64-linux ./nix/systemd-age-creds.nix { };
        x86_64-linux.systemd-age-creds = callPackage.x86_64-linux ./nix/systemd-age-creds.nix { };

        aarch64-linux.default = self.packages.aarch64-linux.systemd-age-creds;
        x86_64-linux.default = self.packages.x86_64-linux.systemd-age-creds;
      };

      overlays.default = final: _prev: {
        systemd-age-creds = final.callPackage ./nix/systemd-age-creds.nix { };
      };

      devShells = {
        aarch64-linux.default = callPackage.aarch64-linux ./nix/shell.nix { };
        x86_64-linux.default = callPackage.x86_64-linux ./nix/shell.nix { };
      };

      nixosModules.default = {
        imports = [ ./nix/nixos.nix ];
        nixpkgs.overlays = [ self.overlays.default ];
      };

      checks = {
        aarch64-linux =
          let
            pkgs = nixpkgs.legacyPackages.aarch64-linux;
            buildPkg = pkg: pkgs.runCommand "${pkg.name}-build" { env.PKG = pkg; } "touch $out";
          in
          {
            systemd-age-creds = buildPkg self.packages.aarch64-linux.systemd-age-creds;
          };

        x86_64-linux =
          let
            pkgs = nixpkgs.legacyPackages.x86_64-linux;
            buildPkg = pkg: pkgs.runCommand "${pkg.name}-build" { env.PKG = pkg; } "touch $out";
          in
          {
            systemd-age-creds = buildPkg self.packages.x86_64-linux.systemd-age-creds;
          }
          // (mapListToAttrs (
            test:
            let
              testName = "nixos-system-unit-${test.creds.name}-creds-accept-${test.accept.name}";
            in
            {
              name = testName;
              value = buildPkg (
                callPackage.x86_64-linux ./nix/tests/nixos-system-unit.nix {
                  inherit self testName;
                  creds = test.creds.value;
                  socketAccept = test.accept.value;
                }
              );
            }
          ) checkMatrix);
      };
    };
}
