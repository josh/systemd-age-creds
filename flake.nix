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
          nixpkgs.legacyPackages.aarch64-linux.callPackage ./systemd-age-creds.nix
            { };
        x86_64-linux.systemd-age-creds =
          nixpkgs.legacyPackages.x86_64-linux.callPackage ./systemd-age-creds.nix
            { };

        aarch64-linux.default = self.packages.aarch64-linux.systemd-age-creds;
        x86_64-linux.default = self.packages.x86_64-linux.systemd-age-creds;
      };

      nixosModules.default = (
        {
          config,
          pkgs,
          lib,
          ...
        }:
        let
          cfg = config.systemd-age-creds;
        in
        {
          options.systemd-age-creds = {
            enable = lib.mkEnableOption "Enable age credentials service";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.system}.default;
              defaultText = lib.literalExpression "pkgs.systemd-age-creds";
              description = "The package to use for systemd-age-creds.";
            };
          };

          config = lib.mkIf cfg.enable {
            systemd.sockets.age-creds = {
              description = "age credentials socket";
              wantedBy = [ "sockets.target" ];

              socketConfig = {
                ListenStream = "/run/age-creds.socket";
                SocketMode = "0600";
                Service = "age-creds.service";
              };
            };

            systemd.services.age-creds = {
              description = "age credentials service";
              requires = [ "age-creds.socket" ];
              serviceConfig = {
                Type = "simple";
                ExecStart = "${lib.getExe cfg.package}";
              };
            };
          };
        }
      );

      checks =
        let
          pkgs = nixpkgs.legacyPackages.x86_64-linux;
        in
        {
          x86_64-linux.nixos = pkgs.testers.runNixOSTest {
            name = "nixos";
            nodes.machine = {
              imports = [ self.nixosModules.default ];
              systemd-age-creds.enable = true;
              systemd.services.age-creds-test = {
                wantedBy = [ "multi-user.target" ];
                serviceConfig = {
                  RemainAfterExit = "yes";
                  LoadCredential = "foo:/run/age-creds.socket";
                  ExecStart = "${pkgs.coreutils}/bin/cp %d/foo /root/foo";
                };
              };
            };
            testScript = ''
              machine.wait_for_unit("age-creds-test.service");
              machine.succeed("test -f /root/foo")
              machine.succeed("test $(cat /root/foo) -eq 42")
            '';
          };
        };
    };
}
