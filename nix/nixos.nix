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

    directory = lib.mkOption {
      type = lib.types.path;
      description = "The directory to load age credentials from.";
    };

    socket = lib.mkOption {
      type = lib.types.path;
      default = "/run/age-creds.sock";
      description = "The path to the age credentials unix socket.";
    };

    socketAccept = lib.mkEnableOption {
      default = false;
      description = "Enable accepting connections on the socket.";
    };
  };

  config =
    let
      serviceName = if cfg.socketAccept then "age-creds@" else "age-creds";
    in
    lib.mkIf cfg.enable {
      systemd.sockets.age-creds = {
        description = "age credentials socket";
        wantedBy = [ "sockets.target" ];

        socketConfig = {
          ListenStream = cfg.socket;
          SocketMode = "0600";
          Accept = if cfg.socketAccept then "yes" else "no";
        };
      };

      systemd.services.${serviceName} = {
        description = "age credentials service";
        serviceConfig = {
          Type = "simple";
          ExecStart = "${lib.getExe cfg.package} ${cfg.directory}";
        };
      };
    };
}
