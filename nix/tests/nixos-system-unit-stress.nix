{
  self,
  testers,
  count ? 25,
}:
testers.runNixOSTest {
  name = "nixos-system-unit-stress";

  node.pkgsReadOnly = false;

  nodes.machine =
    {
      lib,
      config,
      pkgs,
      ...
    }:
    {
      imports = [ self.nixosModules.default ];
      services.systemd-age-creds.enable = true;
      services.systemd-age-creds.directory = ./credstore.encrypted;
      systemd.services.age-creds-test = {
        wantedBy = [ "multi-user.target" ];
        serviceConfig = {
          RemainAfterExit = "yes";
          LoadCredential = lib.genList (
            i: "foo${builtins.toString i}:${config.services.systemd-age-creds.socket}"
          ) count;
          ExecStart = "${pkgs.bash}/bin/bash -c '${pkgs.coreutils}/bin/cp -r %d/foo* /root/'";
        };
      };
    };

  testScript = ''
    machine.wait_for_unit("age-creds-test.service");
    print(machine.succeed("journalctl -u age-creds.socket"))
    print(machine.succeed("journalctl -u age-creds.service"))
    print(machine.succeed("journalctl -u age-creds-test.service"))

    files = machine.succeed("ls /root/foo*").split("\n")
    expected_count = ${builtins.toString count}
    actual_count = len(files)
    assert actual_count == expected_count, f"Expected {expected_count} files, got {actual_count}"
    assert "/root/foo0" in files, "Expected file foo0"
    assert f"/root/foo{expected_count - 1}" in files, f"Expected file foo{expected_count - 1}"

    contents = machine.succeed("cat /root/foo0")
    assert contents == "42\n", f"Expected foo0 to equal '42', got '{contents}'"
  '';
}
