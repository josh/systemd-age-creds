{ lib, buildGoModule }:
buildGoModule {
  pname = "systemd-age-creds";
  version = "0.0.0";
  src = ./.;
  vendorHash = "sha256-NAP44J1nSgjTMlcoT0eCFPEqFTbwmPGe2DqsRhrmAyU=";

  meta = {
    description = "Load age encrypted credentials in systemd units";
    mainProgram = "systemd-age-creds";
    homepage = "https://github.com/josh/systemd-age-creds";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
  };
}
