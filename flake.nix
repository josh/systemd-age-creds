{
  description = "systemd-age-creds";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
  };

  outputs =
    { self, nixpkgs }:
    let
      callPackage = {
        aarch64-linux = nixpkgs.legacyPackages.aarch64-linux.callPackage;
        x86_64-linux = nixpkgs.legacyPackages.x86_64-linux.callPackage;
      };
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
        aarch64-linux.systemd-age-creds = self.packages.aarch64-linux.systemd-age-creds;
        x86_64-linux.systemd-age-creds = self.packages.x86_64-linux.systemd-age-creds;

        x86_64-linux.nixos-system-unit = callPackage.x86_64-linux ./nix/tests/nixos-system-unit.nix {
          inherit self;
        };
        x86_64-linux.nixos-system-unit-stress =
          callPackage.x86_64-linux ./nix/tests/nixos-system-unit-stress.nix
            {
              inherit self;
            };
      };
    };
}
