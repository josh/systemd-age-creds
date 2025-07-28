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

    agePackage = lib.mkOption {
      type = lib.types.nullOr lib.types.package;
      default = null;
      description = "The package to use for age.";
    };

    ageBin = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = if cfg.agePackage != null then (lib.getExe cfg.agePackage) else null;
      description = "The path to the age binary.";
    };

    identity = lib.mkOption {
      type = lib.types.path;
      description = "The path to the age decryption identity.";
    };

    directory = lib.mkOption {
      type = lib.types.path;
      description = "The directory to load age credentials from.";
    };

    acceptTimeout = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      example = "10s";
      description = "Connection handling timeout.";
    };

    idleTimeout = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      example = "5m";
      description = "The time before exiting when there are no connections.";
    };

    socket = lib.mkOption {
      type = lib.types.path;
      default = "/run/systemd-age-creds.sock";
      description = "The path to the age credentials unix socket.";
    };

    socketAccept = lib.mkEnableOption {
      default = false;
      description = "Enable accepting connections on the socket.";
    };
  };

  config =
    let
      serviceName = if cfg.socketAccept then "systemd-age-creds@" else "systemd-age-creds";
    in
    lib.mkIf cfg.enable {
      systemd.sockets.systemd-age-creds = {
        description = "systemd age credentials socket";
        wantedBy = [ "sockets.target" ];

        socketConfig = {
          ListenStream = cfg.socket;
          SocketMode = "0600";
          Accept = if cfg.socketAccept then "yes" else "no";
        };
      };

      systemd.services.${serviceName} = {
        description = "systemd age credentials service";

        unitConfig = {
          AssertFileNotEmpty = cfg.identity;
          AssertDirectoryNotEmpty = cfg.directory;
        };

        serviceConfig = {
          Type = "simple";
          Environment = [
            "AGE_DIR=${cfg.directory}"
            "AGE_IDENTITY=${cfg.identity}"
          ]
          ++ (lib.lists.optional (cfg.ageBin != null) "AGE_BIN=${cfg.ageBin}")
          ++ (lib.lists.optional (cfg.acceptTimeout != null) "AGE_ACCEPT_TIMEOUT=${cfg.acceptTimeout}")
          ++ (lib.lists.optional (cfg.idleTimeout != null) "AGE_IDLE_TIMEOUT=${cfg.idleTimeout}");
          ExecStart = "${lib.getExe cfg.package}";
        };
      };
    };
}
