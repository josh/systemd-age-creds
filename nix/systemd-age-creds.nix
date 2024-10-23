{ lib, buildGoModule }:
buildGoModule {
  pname = "systemd-age-creds";
  version = "0.0.0";
  src = lib.sources.sourceByRegex ./.. [
    ".*\.go$"
    "^go.mod$"
    "^go.sum$"
  ];
  vendorHash = null;

  CGO_ENABLED = 0;

  meta = {
    description = "Load age encrypted credentials in systemd units";
    mainProgram = "systemd-age-creds";
    homepage = "https://github.com/josh/systemd-age-creds";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
  };
}
