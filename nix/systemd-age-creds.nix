{
  lib,
  buildGoModule,
  age,
}:
buildGoModule (finalAttrs: {
  pname = "systemd-age-creds";
  version = "1.0.1";
  src = lib.sources.sourceByRegex ./.. [
    ".*\.go$"
    "^go.mod$"
    "^go.sum$"
    "^systemd$"
    "^systemd\/.*$"
    "^test$"
    "^test\/.*$"
  ];
  vendorHash = null;

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-w"
    "-X main.Version=${finalAttrs.version}"
    "-X main.AgeBin=${lib.getExe age}"
  ];

  nativeBuildInputs = [ age ];

  postInstall = ''
    substituteInPlace ./systemd/*.service --replace-fail /usr/sbin/systemd-age-creds $out/bin/systemd-age-creds
    install -D --mode=0444 --target-directory $out/lib/systemd/system ./systemd/*
  '';

  meta = {
    description = "Load age encrypted credentials in systemd units";
    mainProgram = "systemd-age-creds";
    homepage = "https://github.com/josh/systemd-age-creds";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
  };
})
