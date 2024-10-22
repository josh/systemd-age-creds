{
  config,
  pkgs,
  lib,
  ...
}:
let
  cfg = config.services.systemd-age-creds;
in
{
  options.services.systemd-age-creds = {
    enable = lib.mkEnableOption "Enable age credentials service";

    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.systemd-age-creds;
      defaultText = lib.literalExpression "pkgs.systemd-age-creds";
      description = "The package to use for systemd-age-creds.";
    };

    socket = lib.mkOption {
      type = lib.types.path;
      default = "/run/age-creds.sock";
      description = "The path to the age credentials unix socket.";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.sockets.age-creds = {
      description = "age credentials socket";
      wantedBy = [ "sockets.target" ];

      socketConfig = {
        ListenStream = cfg.socket;
        SocketMode = "0600";
        Service = "age-creds.service";
      };
    };

    systemd.services.age-creds = {
      description = "age credentials service";
      requires = [ "age-creds.socket" ];
      serviceConfig = {
        Type = "simple";
        ExecStart = "${lib.getExe cfg.package}";
      };
    };
  };
}
