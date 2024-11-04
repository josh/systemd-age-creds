{ self, testers }:
testers.runNixOSTest {
  name = "nixos-system-unit";

  node.pkgsReadOnly = false;

  nodes.machine =
    { config, pkgs, ... }:
    {
      imports = [ self.nixosModules.default ];
      services.systemd-age-creds.enable = true;
      services.systemd-age-creds.identity = ./key.txt;
      services.systemd-age-creds.directory = ./credstore.encrypted;
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
    print(machine.succeed("journalctl -u age-creds.socket"))
    print(machine.succeed("journalctl -u age-creds.service"))
    print(machine.succeed("journalctl -u age-creds-test.service"))

    contents = machine.succeed("cat /root/foo").strip()
    assert contents == "42", f"Expected foo to equal '42', got '{contents}'"
  '';
}
