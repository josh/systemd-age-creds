{
  lib,
  buildGoModule,
  age,
}:
let
  version = "0.0.0";
in
buildGoModule {
  pname = "systemd-age-creds";
  version = version;
  src = lib.sources.sourceByRegex ./.. [
    ".*\.go$"
    "^go.mod$"
    "^go.sum$"
  ];
  vendorHash = null;

  CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-w"
    "-X main.Version=${version}"
    "-X main.AGE_BIN=${lib.getExe age}"
  ];

  meta = {
    description = "Load age encrypted credentials in systemd units";
    mainProgram = "systemd-age-creds";
    homepage = "https://github.com/josh/systemd-age-creds";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
  };
}
