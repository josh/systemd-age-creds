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

      checks =
        let
          oneCred = {
            foo = "42";
          };
          fewCreds = builtins.listToAttrs (
            builtins.genList (i: {
              name = "foo-${builtins.toString i}";
              value = builtins.toString i;
            }) 3
          );
          manyCreds = builtins.listToAttrs (
            builtins.genList (i: {
              name = "foo-${builtins.toString i}";
              value = builtins.toString i;
            }) 50
          );
        in
        {
          aarch64-linux.systemd-age-creds = self.packages.aarch64-linux.systemd-age-creds;
          x86_64-linux.systemd-age-creds = self.packages.x86_64-linux.systemd-age-creds;

          x86_64-linux.nixos-system-unit-one-accept-no =
            callPackage.x86_64-linux ./nix/tests/nixos-system-unit.nix
              {
                inherit self;
                testName = "accept-no";
                creds = oneCred;
                socketAccept = false;
              };
          x86_64-linux.nixos-system-unit-one-accept-yes =
            callPackage.x86_64-linux ./nix/tests/nixos-system-unit.nix
              {
                inherit self;
                testName = "accept-yes";
                creds = oneCred;
                socketAccept = true;
              };

          x86_64-linux.nixos-system-unit-few-accept-no =
            callPackage.x86_64-linux ./nix/tests/nixos-system-unit.nix
              {
                inherit self;
                testName = "few-accept-no";
                creds = fewCreds;
                socketAccept = false;
              };
          # x86_64-linux.nixos-system-unit-few-accept-yes =
          #   callPackage.x86_64-linux ./nix/tests/nixos-system-unit.nix
          #     {
          #       inherit self;
          #       testName = "few-accept-yes";
          #       creds = fewCreds;
          #       socketAccept = true;
          #     };

          x86_64-linux.nixos-system-unit-many-accept-no =
            callPackage.x86_64-linux ./nix/tests/nixos-system-unit.nix
              {
                testName = "many-accept-no";
                inherit self;
                creds = manyCreds;
                socketAccept = false;
              };
        };
    };
}
