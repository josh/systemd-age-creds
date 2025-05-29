{
  lib,
  self,
  runCommandLocal,
  testers,
  writeShellScript,
  age,
  coreutils,
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
          echo "${value}" | age -R ${keys}/recipients >$out/${name}.age
        '') creds);
      script = lib.concatStringsSep "\n" commands;
    in
    runCommandLocal "credstore" { buildInputs = [ age ]; } script;

  keys = runCommandLocal "age-keygen" { buildInputs = [ age ]; } ''
    mkdir $out
    age-keygen -o $out/identity
    age-keygen -y -o $out/recipients $out/identity
  '';

  copyCredsScript = writeShellScript "export-creds.bash" ''
    ${coreutils}/bin/mkdir /tmp/systemd-age-creds-test
    ${coreutils}/bin/cp $CREDENTIALS_DIRECTORY/* /tmp/systemd-age-creds-test/
  '';

in

assert credCount > 0;

testers.runNixOSTest {
  name = testName;

  node.pkgsReadOnly = false;

  nodes.machine =
    { config, ... }:
    {
      imports = [ self.nixosModules.default ];

      services.systemd-age-creds = {
        enable = true;
        identity = "${keys}/identity";
        directory = credstoreDir;
        inherit socketAccept;
      };

      systemd.services.systemd-age-creds-test = {
        wantedBy = [ "multi-user.target" ];
        serviceConfig = {
          Type = "oneshot";
          RemainAfterExit = "yes";
          LoadCredential = builtins.map (
            name: "${name}:${config.services.systemd-age-creds.socket}"
          ) credNames;
          ExecStart = copyCredsScript;
        };
      };
    };

  testScript = ''
    import json
    creds = json.loads('${builtins.toJSON creds}')
    accept = ${if socketAccept then "True" else "False"}

    machine.wait_for_unit("systemd-age-creds.socket");
    machine.wait_for_unit("systemd-age-creds-test.service");

    if accept:
      n_connections = int(machine.get_unit_property("systemd-age-creds.socket", "NConnections"))
      n_accepted = int(machine.get_unit_property("systemd-age-creds.socket", "NAccepted"))
      assert n_connections == 0, f"Expected 0 active connection, got {n_connections}"
      assert n_accepted == len(creds), f"Expected {len(creds)} accepted connections, got {n_accepted}"

    files = machine.succeed("ls /tmp/systemd-age-creds-test").splitlines()
    assert len(files) > 0, "Expected at least one file in /tmp/systemd-age-creds-test"
    assert len(files) == len(creds), f"Expected {len(creds)} files, got {len(files)}: {files}"

    for name, value in creds.items():
      assert name in files, f"Expected file {name}"
      contents = machine.succeed(f"cat /tmp/systemd-age-creds-test/{name}").strip()
      assert contents == value, f"Expected {name} to equal '{value}', got '{contents}'"
  '';
}
