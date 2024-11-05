{
  lib,
  self,
  age,
  runCommandLocal,
  testers,
  testName,
  creds,
  socketAccept ? false,
}:
let
  credNames = builtins.attrNames creds;
  credCount = builtins.length credNames;

  credstoreDir =
    let
      commands =
        [ "mkdir $out" ]
        ++ (lib.attrsets.mapAttrsToList (name: value: ''
          echo "${value}" >$out/${name}.age
        '') creds);
      script = lib.concatStringsSep "\n" commands;
    in
    runCommandLocal "credstore" { } script;

  identityFile =
    runCommandLocal "identity"
      {
        buildInputs = [ age ];
      }
      ''
        age-keygen -o $out
      '';
in

assert credCount > 0;

testers.runNixOSTest {
  name = testName;

  node.pkgsReadOnly = false;

  nodes.machine =
    {
      config,
      pkgs,
      ...
    }:
    {
      imports = [ self.nixosModules.default ];
      services.systemd-age-creds = {
        enable = true;
        identity = identityFile;
        directory = credstoreDir;
        inherit socketAccept;
      };
      systemd.services.age-creds-test = {
        wantedBy = [ "multi-user.target" ];
        serviceConfig = {
          Type = "oneshot";
          RemainAfterExit = "yes";
          LoadCredential = builtins.map (
            name: "${name}:${config.services.systemd-age-creds.socket}"
          ) credNames;
          ExecStart = "${pkgs.bash}/bin/bash -c '${pkgs.coreutils}/bin/cp -r %d/* /root/'";
        };
      };
    };

  testScript = ''
    import json
    creds = json.loads('${builtins.toJSON creds}')
    accept = ${if socketAccept then "True" else "False"}

    machine.wait_for_unit("age-creds.socket");
    machine.wait_for_unit("age-creds-test.service");
    print(machine.succeed("journalctl -u age-creds.socket"))
    print(machine.succeed("journalctl -u age-creds.service"))
    print(machine.succeed("journalctl -u age-creds-test.service"))

    if accept:
      n_connections = int(machine.get_unit_property("age-creds.socket", "NConnections"))
      n_accepted = int(machine.get_unit_property("age-creds.socket", "NAccepted"))
      assert n_connections == 0, f"Expected 0 active connection, got {n_connections}"
      assert n_accepted == len(creds), f"Expected {len(creds)} accepted connections, got {n_accepted}"

    files = machine.succeed("ls /root").splitlines()
    assert len(files) > 0, "Expected at least one file in /root"
    assert len(files) == len(creds), f"Expected {len(creds)} files, got {len(files)}: {files}"

    for name, value in creds.items():
      assert name in files, f"Expected file {name}"
      contents = machine.succeed(f"cat /root/{name}").strip()
      assert contents == value, f"Expected {name} to equal '{value}', got '{contents}'"
  '';
}
