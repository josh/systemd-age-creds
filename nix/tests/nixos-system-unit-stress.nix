{
  self,
  runCommandLocal,
  testers,
  count ? 25,
}:
let
  credstoreDir = runCommandLocal "credstore" { } ''
    mkdir -p $out
    for i in $(seq 1 ${builtins.toString count}); do
      cp ${./credstore.encrypted/foo.age} $out/foo-$i.age
    done
  '';
in
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
      services.systemd-age-creds.identity = ./key.txt;
      services.systemd-age-creds.directory = credstoreDir;
      systemd.services.age-creds-test = {
        wantedBy = [ "multi-user.target" ];
        serviceConfig = {
          RemainAfterExit = "yes";
          LoadCredential = lib.genList (
            i: "foo-${builtins.toString (i + 1)}:${config.services.systemd-age-creds.socket}"
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

    files = machine.succeed("ls /root").splitlines()
    expected_count = ${builtins.toString count}
    actual_count = len(files)
    assert actual_count == expected_count, f"Expected {expected_count} files, got {actual_count}: {files}"
    assert "foo-1" in files, "Expected file foo-1"
    assert f"foo-{expected_count}" in files, f"Expected file foo-{expected_count}"

    contents = machine.succeed("cat /root/foo-1").strip()
    assert contents == "42", f"Expected foo-1 to equal '42', got '{contents}'"
  '';
}
