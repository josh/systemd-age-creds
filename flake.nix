{
  description = "systemd-age-creds";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/24.05";
  };

  outputs =
    { self, nixpkgs }:
    {
      packages = {
        aarch64-linux.systemd-age-creds =
          nixpkgs.legacyPackages.aarch64-linux.callPackage ./nix/systemd-age-creds.nix
            { };
        x86_64-linux.systemd-age-creds =
          nixpkgs.legacyPackages.x86_64-linux.callPackage ./nix/systemd-age-creds.nix
            { };

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
          runNixOSTest = nixpkgs.legacyPackages.x86_64-linux.testers.runNixOSTest;
        in
        {
          aarch64-linux.systemd-age-creds = self.packages.aarch64-linux.systemd-age-creds;
          x86_64-linux.systemd-age-creds = self.packages.x86_64-linux.systemd-age-creds;

          x86_64-linux.nixos-system-unit = runNixOSTest {
            name = "nixos-system-unit";
            node.pkgsReadOnly = false;
            nodes.machine = (
              { config, pkgs, ... }:
              {
                imports = [ self.nixosModules.default ];
                services.systemd-age-creds.enable = true;
                systemd.services.age-creds-test = {
                  wantedBy = [ "multi-user.target" ];
                  serviceConfig = {
                    RemainAfterExit = "yes";
                    LoadCredential = "foo:${config.services.systemd-age-creds.socket}";
                    ExecStart = "${pkgs.coreutils}/bin/cp %d/foo /root/foo";
                  };
                };
              }
            );
            testScript = ''
              machine.wait_for_unit("age-creds-test.service");
              machine.succeed("test -f /root/foo")
              machine.succeed("test $(cat /root/foo) -eq 42")
            '';
          };
        };
    };
}
