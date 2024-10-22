{
  self,
  testers,
  count ? 25,
}:
testers.runNixOSTest {
  name = "nixos-system-unit";

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
    machine.succeed("test -f /root/foo0")
    machine.succeed("test -f /root/foo${builtins.toString (count - 1)}")
    machine.succeed("test $(ls /root/foo* | wc -l) -eq ${builtins.toString count}")
    machine.succeed("test $(cat /root/foo0) -eq 42")
  '';
}
