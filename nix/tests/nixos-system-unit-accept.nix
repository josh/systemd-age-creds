{ self, testers }:
testers.runNixOSTest {
  name = "nixos-system-unit-accept";

  node.pkgsReadOnly = false;

  nodes.machine =
    { config, pkgs, ... }:
    {
      imports = [ self.nixosModules.default ];
      services.systemd-age-creds.enable = true;
      services.systemd-age-creds.directory = ./credstore.encrypted;
      services.systemd-age-creds.socketAccept = true;
      systemd.services.age-creds-test = {
        wantedBy = [ "multi-user.target" ];
        serviceConfig = {
          RemainAfterExit = "yes";
          LoadCredential = "foo:${config.services.systemd-age-creds.socket}";
          ExecStart = "${pkgs.coreutils}/bin/cp %d/foo /root/foo";
        };
      };
    };

  testScript = ''
    machine.wait_for_unit("age-creds-test.service");
    machine.succeed("test -f /root/foo")
    machine.succeed("test $(cat /root/foo) -eq 42")
  '';
}
