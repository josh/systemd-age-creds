{
  lib,
  buildGoModule,
  age,
}:
let
  version = "0.2.0";
in
buildGoModule {
  pname = "systemd-age-creds";
  inherit version;
  src = lib.sources.sourceByRegex ./.. [
    ".*\.go$"
    "^go.mod$"
    "^go.sum$"
    "^test$"
    "^test\/.*$"
  ];
  vendorHash = null;

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-w"
    "-X main.Version=${version}"
    "-X main.AgeBin=${lib.getExe age}"
  ];

  nativeBuildInputs = [ age ];

  meta = {
    description = "Load age encrypted credentials in systemd units";
    mainProgram = "systemd-age-creds";
    homepage = "https://github.com/josh/systemd-age-creds";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
  };
}
