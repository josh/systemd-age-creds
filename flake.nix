{
  description = "systemd-age-creds";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
  };

  outputs =
    { self, nixpkgs }:
    let
      lib = nixpkgs.lib;
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

      overlays.default = final: prev: {
        systemd-age-creds = final.callPackage ./nix/systemd-age-creds.nix { };
      };

      nixosModules.default = {
        imports = [ ./nix/nixos.nix ];
        nixpkgs.overlays = [ self.overlays.default ];
      };

      checks = {
        aarch64-linux = {
          systemd-age-creds = self.packages.aarch64-linux.systemd-age-creds;
        };

        x86_64-linux =
          {
            systemd-age-creds = self.packages.x86_64-linux.systemd-age-creds;
          }
          // (mapListToAttrs (
            test:
            let
              testName = "nixos-system-unit-${test.creds.name}-creds-accept-${test.accept.name}";
            in
            {
              name = testName;
              value = callPackage.x86_64-linux ./nix/tests/nixos-system-unit.nix {
                inherit self testName;
                creds = test.creds.value;
                socketAccept = test.accept.value;
              };
            }
          ) checkMatrix);
      };
    };
}
